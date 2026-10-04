package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

type backfillChoice struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Filenames []string `json:"filenames"`
	Error     string   `json:"error,omitempty"`
	source    feed.DownloadChoice
}
type backfillPlan struct {
	mu               sync.Mutex
	until            time.Time
	account, session string
	sub              model.Subscription
	choices          []backfillChoice
	selection        string
	jobs             []model.Job
}

func (s *Server) previewBackfill(w http.ResponseWriter, r *http.Request) {
	var in struct{}
	if err := decode(w, r, &in); err != nil {
		failure(w, err)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		failure(w, errors.New("訂閱 ID 無效"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	_, account, err := s.directoryAPI()
	if err != nil {
		failure(w, err)
		return
	}
	sub, err := s.DB.Subscription(ctx, id)
	if err != nil {
		failure(w, errors.New("找不到項目"))
		return
	}
	if sub.DestinationID != "" && sub.DestinationAccountID != account {
		failure(w, errors.New("PikPak 帳號已更換，請重新選取此訂閱的目標資料夾"))
		return
	}
	client := feed.New(s.appSettings().AllowPrivateFeeds)
	defer client.HTTP.CloseIdleConnections()
	sources, notices, err := client.DownloadChoices(ctx, sub.RSSURL)
	if err != nil {
		failure(w, err)
		return
	}
	choices := make([]backfillChoice, 0, len(sources))
	outputBytes := 0
	for _, source := range sources {
		choice := backfillChoice{ID: store.ID(), Title: source.Item.Title, Filenames: source.Filenames, Error: source.Error, source: source}
		encoded, _ := json.Marshal(choice)
		outputBytes += len(encoded) + 1
		if outputBytes > (8<<20)-4096 {
			notices = append(notices, "檔名清單超過 10,000 筆或 8 MiB，已保留部分檔名；請使用較小的 RSS。")
			break
		}
		choices = append(choices, choice)
	}
	cookie, _ := r.Cookie("pp_session")
	plan := &backfillPlan{until: time.Now().Add(15 * time.Minute), account: account, session: cookie.Value, sub: sub, choices: choices}
	token := nonce()
	s.backfillMu.Lock()
	if s.backfills == nil {
		s.backfills = map[string]*backfillPlan{}
	}
	for key, p := range s.backfills {
		if time.Now().After(p.until) {
			delete(s.backfills, key)
		}
	}
	if len(s.backfills) >= 4 {
		var oldest string
		for key, p := range s.backfills {
			if oldest == "" || p.until.Before(s.backfills[oldest].until) {
				oldest = key
			}
		}
		delete(s.backfills, oldest)
	}
	s.backfills[token] = plan
	s.backfillMu.Unlock()
	JSON(w, 200, map[string]any{"token": token, "items": choices, "notices": notices})
}

func (s *Server) confirmBackfill(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token     string   `json:"token"`
		Selected  []string `json:"selected"`
		Overwrite bool     `json:"overwrite"`
	}
	if err := decode(w, r, &in); err != nil {
		failure(w, err)
		return
	}
	if len(in.Selected) == 0 || len(in.Selected) > 2000 {
		failure(w, errors.New("請選擇下載項目"))
		return
	}
	s.backfillMu.Lock()
	plan := s.backfills[in.Token]
	s.backfillMu.Unlock()
	cookie, _ := r.Cookie("pp_session")
	if plan == nil || cookie == nil || plan.session != cookie.Value || time.Now().After(plan.until) {
		failure(w, errors.New("補抓清單已失效，請重新讀取"))
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id != plan.sub.ID {
		failure(w, errors.New("補抓清單已失效，請重新讀取"))
		return
	}
	plan.mu.Lock()
	defer plan.mu.Unlock()
	selected := map[string]bool{}
	for _, id := range in.Selected {
		if selected[id] {
			failure(w, errors.New("補抓選取項目無效"))
			return
		}
		selected[id] = true
	}
	sort.Strings(in.Selected)
	selection := strings.Join(in.Selected, ",")
	if plan.selection != "" && plan.selection != selection {
		failure(w, errors.New("此補抓清單已送出，請重新讀取"))
		return
	}
	s.Worker.Gate.Lock()
	defer s.Worker.Gate.Unlock()
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	_, account, err := s.directoryAPI()
	if err != nil {
		failure(w, err)
		return
	}
	if account != plan.account {
		failure(w, errors.New("PikPak 帳號已更換，請重新讀取補抓清單"))
		return
	}
	if plan.selection != "" {
		JSON(w, 200, map[string]any{"jobs": plan.jobs, "count": len(plan.jobs)})
		return
	}
	sub := plan.sub
	sub.DestinationAccountRef = s.accountReference(account)
	if sub.DestinationID != "" {
		if err := s.resolveDestinationLocked(ctx, &sub); err != nil {
			failure(w, err)
			return
		}
	}
	jobs := []model.Job{}
	fingerprints := []string{}
	now := time.Now().Unix()
	for _, choice := range plan.choices {
		if !selected[choice.ID] {
			continue
		}
		if choice.Error != "" || choice.source.Resource.Key == "" {
			failure(w, errors.New("補抓選取項目無效"))
			return
		}
		delete(selected, choice.ID)
		jobs = append(jobs, model.Job{DownloadMode: "direct", ID: choice.ID, SubscriptionID: sub.ID, AccountID: account, ResourceKey: choice.source.Resource.Key, ResourceURL: choice.source.Resource.URL, Title: choice.Title, Rule: sub.Rule(), Destination: sub.Destination, DestinationID: sub.DestinationID, State: "queued", NextAttempt: now, CreatedAt: now, UpdatedAt: now})
		fingerprints = append(fingerprints, choice.source.Item.Fingerprint)
	}
	if len(selected) > 0 {
		failure(w, errors.New("補抓選取項目無效"))
		return
	}
	if err := s.Worker.QueueSelection(ctx, plan.sub, jobs, fingerprints); err != nil {
		failure(w, err)
		return
	}
	plan.selection = selection
	plan.jobs = jobs
	for _, job := range jobs {
		_ = s.DB.Event(ctx, job.ID, sub.ID, "info", "已確認下載，離線任務已排入佇列")
	}
	JSON(w, 200, map[string]any{"jobs": jobs, "count": len(jobs)})
}
