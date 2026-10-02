package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/rename"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"path"
	"strings"
	"sync"
	"time"
)

type Provider interface {
	Snapshot() (pikpak.API, string)
	Pause(*pikpak.Error)
}
type Feeds interface {
	Fetch(context.Context, string) ([]feed.Item, error)
	Resolve(context.Context, string) (feed.Resource, error)
}
type Worker struct {
	DB          *store.Store
	Provider    Provider
	Feeds       Feeds
	StagingPath string
	Gate        sync.Mutex
	subMu       sync.Mutex
}

func New(db *store.Store, p Provider, f Feeds) *Worker { return &Worker{DB: db, Provider: p, Feeds: f} }
func (w *Worker) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			w.checkDue(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			w.processDue(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	<-ctx.Done()
	wg.Wait()
}
func (w *Worker) checkDue(ctx context.Context) {
	subs, err := w.DB.Subscriptions(ctx)
	if err != nil {
		return
	}
	now := time.Now().Unix()
	for _, sub := range subs {
		if ctx.Err() != nil {
			return
		}
		if sub.Enabled && sub.NextCheck <= now {
			_ = w.Check(ctx, sub.ID, false)
		}
	}
}
func (w *Worker) processDue(ctx context.Context) {
	api, account := w.Provider.Snapshot()
	if api == nil {
		return
	}
	jobs, err := w.DB.DueJobs(ctx, account, time.Now().Unix())
	if err != nil {
		return
	}
	for _, job := range jobs {
		if ctx.Err() != nil {
			return
		}
		_ = w.Process(ctx, job.ID)
	}
}
func (w *Worker) SaveSubscription(ctx context.Context, sub *model.Subscription) error {
	w.subMu.Lock()
	defer w.subMu.Unlock()
	if sub.ID != 0 {
		old, err := w.DB.Subscription(ctx, sub.ID)
		if err != nil {
			return err
		}
		sub.Initialized = old.Initialized
		sub.LastChecked = old.LastChecked
		sub.LastError = old.LastError
		if old.RSSURL != sub.RSSURL {
			sub.Initialized = false
			if err := w.DB.ResetFeed(ctx, sub.ID); err != nil {
				return err
			}
		}
	}
	sub.NextCheck = time.Now().Unix()
	return w.DB.SaveSubscription(ctx, sub)
}
func (w *Worker) DeleteSubscription(ctx context.Context, id int64) error {
	w.subMu.Lock()
	defer w.subMu.Unlock()
	return w.DB.DeleteSubscription(ctx, id)
}
func (w *Worker) Check(ctx context.Context, id int64, backfill bool) error {
	w.subMu.Lock()
	defer w.subMu.Unlock()
	sub, err := w.DB.Subscription(ctx, id)
	if err != nil {
		return err
	}
	api, account := w.Provider.Snapshot()
	if api == nil || account == "" {
		return errors.New("請先綁定 PikPak，或解除授權／配額暫停")
	}
	items, err := w.Feeds.Fetch(ctx, sub.RSSURL)
	sub.LastChecked = time.Now().Unix()
	sub.NextCheck = time.Now().Add(time.Duration(sub.IntervalMinutes) * time.Minute).Unix()
	if err != nil {
		sub.LastError = err.Error()
		_ = w.DB.SaveSubscription(ctx, &sub)
		_ = w.DB.Event(ctx, "", id, "error", sub.LastError)
		return err
	}
	sub.LastError = ""
	if !sub.Initialized && !backfill {
		fingerprints := []string{}
		for _, item := range items {
			fingerprints = append(fingerprints, item.Fingerprint)
		}
		if err := w.DB.Baseline(ctx, &sub, fingerprints); err != nil {
			return err
		}
		return w.DB.Event(ctx, "", id, "info", fmt.Sprintf("首次基準已建立：%d 筆既有項目；僅處理之後的新項目", len(items)))
	}
	created, duplicates := 0, 0
	for _, item := range items {
		seen, baseline, err := w.DB.Seen(ctx, id, item.Fingerprint)
		if err != nil {
			return err
		}
		if seen && (!backfill || !baseline) {
			continue
		}
		resource, err := w.Feeds.Resolve(ctx, item.URL)
		if err != nil {
			sub.LastError = err.Error()
			_ = w.DB.Event(ctx, "", id, "warning", err.Error())
			continue
		}
		now := time.Now().Unix()
		j := model.Job{ID: store.ID(), SubscriptionID: id, AccountID: account, ResourceKey: resource.Key, ResourceURL: resource.URL, Title: item.Title, Rule: sub.Rule(), Destination: sub.Destination, State: "queued", NextAttempt: now, CreatedAt: now, UpdatedAt: now}
		added, err := w.DB.Enqueue(ctx, j, item.Fingerprint)
		if err != nil {
			return err
		}
		if added {
			created++
			_ = w.DB.Event(ctx, j.ID, id, "info", "離線任務已排入佇列")
		} else {
			duplicates++
		}
	}
	sub.Initialized = true
	if err := w.DB.SaveSubscription(ctx, &sub); err != nil {
		return err
	}
	return w.DB.Event(ctx, "", id, "info", fmt.Sprintf("檢查完成：新增 %d 筆、去重 %d 筆", created, duplicates))
}

func ensurePath(ctx context.Context, api pikpak.API, parent, destination string) (string, error) {
	if destination == "" {
		return parent, nil
	}
	for _, segment := range strings.Split(destination, "/") {
		id, err := pikpak.EnsureFolder(ctx, api, parent, segment)
		if err != nil {
			return "", err
		}
		parent = id
	}
	return parent, nil
}

func (w *Worker) Process(ctx context.Context, id string) error {
	w.Gate.Lock()
	defer w.Gate.Unlock()
	j, err := w.DB.Job(ctx, id)
	if err != nil {
		return err
	}
	api, account := w.Provider.Snapshot()
	if api == nil {
		return nil
	}
	if account != j.AccountID {
		j.State = "paused_account"
		j.Error = "舊帳號任務已暫停"
		return w.DB.SaveJob(ctx, &j)
	}
	if j.State == "submitting" {
		return w.reconcile(ctx, api, &j)
	}
	if j.State == "queued" {
		if j.DestinationID == "" {
			j.DestinationID, err = ensurePath(ctx, api, "", j.Destination)
			if err != nil {
				return w.failure(ctx, &j, err, false)
			}
		}
		if j.StagingID == "" {
			stagingPath := w.StagingPath
			if stagingPath == "" {
				stagingPath = "_PikPak-RSS-Staging"
			}
			root, e := ensurePath(ctx, api, "", stagingPath)
			if e != nil {
				return w.failure(ctx, &j, e, false)
			}
			j.StagingID, err = pikpak.EnsureFolder(ctx, api, root, j.ID)
			if err != nil {
				return w.failure(ctx, &j, err, false)
			}
		}
		// Persist intent BEFORE calling the non-idempotent cloud operation.
		j.State = "submitting"
		if err := w.DB.SaveJob(ctx, &j); err != nil {
			return err
		}
		task, e := api.Submit(ctx, j.StagingID, j.ResourceURL)
		if e != nil {
			return w.failure(ctx, &j, e, true)
		}
		j.TaskID = task.ID
		j.State = "downloading"
		j.Error = ""
		j.Attempts = 0
		j.NextAttempt = time.Now().Add(15 * time.Second).Unix()
		if err := w.DB.SaveJob(ctx, &j); err != nil {
			return err
		}
		_ = w.DB.Event(ctx, j.ID, j.SubscriptionID, "info", "PikPak 已接受離線任務")
		return nil
	}
	if j.State == "downloading" {
		task, e := api.Task(ctx, j.TaskID)
		if e != nil {
			return w.failure(ctx, &j, e, false)
		}
		j.Progress = task.Percent()
		j.Attempts = 0
		j.Error = ""
		switch task.State() {
		case "complete", "completed":
			j.FileID = task.FileID
			if j.FileID == "" {
				roots, e := pikpak.ListAll(ctx, api, j.StagingID)
				if e != nil {
					return w.failure(ctx, &j, e, false)
				}
				if len(roots) != 1 {
					j.State = "needs_review"
					j.Error = "任務完成但無法唯一定位雲端檔案"
					return w.DB.SaveJob(ctx, &j)
				}
				j.FileID = roots[0].ID
			}
			j.State = "organizing"
			j.Progress = 100
		case "error", "failed":
			if e := pikpak.Classify(errors.New(task.Message)); e.Kind == "auth" || e.Kind == "quota" {
				return w.failure(ctx, &j, e, false)
			}
			j.State = "failed"
			j.Error = "PikPak 離線任務失敗；請查看 PikPak 端的來源可用性"
			_ = w.DB.Event(ctx, j.ID, j.SubscriptionID, "error", j.Error)
			return w.DB.SaveJob(ctx, &j)
		default:
			if time.Now().Unix()-j.CreatedAt > 7*24*3600 {
				j.State = "needs_review"
				j.Error = "離線任務超過七天，請至 PikPak 檢查"
			} else {
				j.NextAttempt = time.Now().Add(30 * time.Second).Unix()
			}
			return w.DB.SaveJob(ctx, &j)
		}
		if err := w.DB.SaveJob(ctx, &j); err != nil {
			return err
		}
	}
	if j.State == "organizing" {
		if err := w.organize(ctx, api, &j); err != nil {
			return w.failure(ctx, &j, err, false)
		}
	}
	return nil
}

func (w *Worker) failure(ctx context.Context, j *model.Job, err error, submitted bool) error {
	e := pikpak.Classify(err)
	j.Error = e.Message
	j.Attempts++
	if submitted && e.Kind != "auth" && e.Kind != "quota" && e.Kind != "rate" {
		j.State = "submission_unknown"
		j.Error = "離線提交結果不明，請重新核對；系統不會再次自動提交"
	} else if e.Kind == "auth" || e.Kind == "quota" {
		if submitted {
			j.State = "queued"
		}
		j.State = "paused_" + e.Kind
		w.Provider.Pause(e)
	} else if e.Kind == "permanent" || j.Attempts >= 6 {
		j.State = "needs_review"
	} else {
		if submitted {
			j.State = "queued"
		}
		delays := []time.Duration{30 * time.Second, 2 * time.Minute, 10 * time.Minute, 30 * time.Minute, 2 * time.Hour}
		i := j.Attempts - 1
		if i >= len(delays) {
			i = len(delays) - 1
		}
		j.NextAttempt = time.Now().Add(delays[i]).Unix()
	}
	if saveErr := w.DB.SaveJob(ctx, j); saveErr != nil {
		return saveErr
	}
	_ = w.DB.Event(ctx, j.ID, j.SubscriptionID, "warning", j.Error)
	return e
}

func (w *Worker) reconcile(ctx context.Context, api pikpak.API, j *model.Job) error {
	roots, err := pikpak.ListAll(ctx, api, j.StagingID)
	if err != nil {
		return err
	}
	if len(roots) == 1 && strings.Contains(strings.ToLower(roots[0].Phase), "complete") {
		j.FileID = roots[0].ID
		j.State = "organizing"
		j.Error = ""
		j.NextAttempt = 0
	} else {
		j.State = "submission_unknown"
		j.Error = "提交結果仍不明；請在 PikPak 檢查此任務的專用暫存目錄，系統不會重複提交"
	}
	return w.DB.SaveJob(ctx, j)
}
func (w *Worker) Retry(ctx context.Context, id string) error {
	w.Gate.Lock()
	defer w.Gate.Unlock()
	j, err := w.DB.Job(ctx, id)
	if err != nil {
		return err
	}
	api, account := w.Provider.Snapshot()
	if api == nil {
		return errors.New("請先重新檢查 PikPak 連線")
	}
	if j.AccountID != account {
		return errors.New("此任務屬於其他 PikPak 帳號")
	}
	if j.State == "complete" {
		return errors.New("任務已完成")
	}
	if j.State == "submission_unknown" || j.State == "submitting" {
		return w.reconcile(ctx, api, &j)
	}
	if j.State == "failed" && j.TaskID != "" {
		return errors.New("遠端離線任務已失敗；請先在 PikPak 處理來源，本系統不重複提交")
	}
	if sub, e := w.DB.Subscription(ctx, j.SubscriptionID); e == nil {
		j.Rule = sub.Rule()
	}
	if j.FileID != "" {
		j.State = "organizing"
	} else if j.TaskID != "" {
		j.State = "downloading"
	} else if j.StagingID != "" && j.State == "paused_account" {
		j.State = "submitting"
	} else {
		j.State = "queued"
	}
	j.Error = ""
	j.Attempts = 0
	j.NextAttempt = 0
	return w.DB.SaveJob(ctx, &j)
}

type entry struct {
	file     pikpak.File
	relative string
}

func walk(ctx context.Context, api pikpak.API, root pikpak.File, prefix string, depth int, out *[]entry) error {
	if depth > 32 || len(*out) > 5000 {
		return &pikpak.Error{Kind: "permanent", Message: "種子目錄超過處理上限"}
	}
	if !root.Folder() {
		*out = append(*out, entry{root, prefix + root.Name})
		return nil
	}
	files, err := pikpak.ListAll(ctx, api, root.ID)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.Folder() {
			if err := walk(ctx, api, f, prefix+f.Name+"/", depth+1, out); err != nil {
				return err
			}
		} else {
			if len(*out) >= 5000 {
				return &pikpak.Error{Kind: "permanent", Message: "種子檔案超過 5000 個上限"}
			}
			*out = append(*out, entry{f, prefix + f.Name})
		}
	}
	return nil
}
func primary(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".mkv", ".mp4", ".avi", ".webm", ".mov", ".m4v", ".ts", ".m2ts", ".wmv":
		return true
	}
	return false
}
func (w *Worker) organize(ctx context.Context, api pikpak.API, j *model.Job) error {
	actions, err := w.DB.Actions(ctx, j.ID)
	if err != nil {
		return err
	}
	known := map[string]model.FileAction{}
	for _, a := range actions {
		known[a.FileID] = a
	}
	root, err := api.Get(ctx, j.FileID)
	if err != nil {
		return err
	}
	entries := []entry{}
	if err := walk(ctx, api, root, "", 0, &entries); err != nil {
		return err
	}
	if j.FileCount == 0 {
		j.FileCount = len(entries)
		for _, e := range entries {
			if primary(e.file.Name) {
				j.PrimaryCount++
			}
		}
		if j.PrimaryCount == 0 && j.FileCount == 1 {
			j.PrimaryCount = 1
		}
		if j.FileCount == 0 {
			return &pikpak.Error{Kind: "permanent", Message: "完成的任務沒有檔案，請至 PikPak 檢查"}
		}
		if err := w.DB.SaveJob(ctx, j); err != nil {
			return err
		}
	}
	for _, e := range entries {
		a, exists := known[e.file.ID]
		if exists && a.State != "review" {
			continue
		}
		if !exists {
			a = model.FileAction{JobID: j.ID, FileID: e.file.ID, OriginalName: e.file.Name, RelativePath: e.relative}
		}
		if !primary(a.OriginalName) && j.FileCount != 1 {
			a.State = "pending"
			a.TargetName = a.OriginalName
			extraPath := "_附件/" + j.ID
			if dir := path.Dir(a.RelativePath); dir != "." {
				extraPath += "/" + dir
			}
			a.DestinationID, err = ensurePath(ctx, api, j.DestinationID, extraPath)
			if err != nil {
				return err
			}
			a.Error = ""
		} else {
			preview, parseErr := rename.Render(j.Rule, j.Title, a.OriginalName, j.PrimaryCount == 1)
			if parseErr != nil {
				a.State = "review"
				a.Error = parseErr.Error()
			} else {
				a.State = "pending"
				a.TargetName = preview.Name
				a.DestinationID = j.DestinationID
				a.Error = ""
			}
		}
		if err := w.DB.SaveAction(ctx, a); err != nil {
			return err
		}
		known[a.FileID] = a
	}
	// The complete action plan exists before the first rename or move.
	actions, err = w.DB.Actions(ctx, j.ID)
	if err != nil {
		return err
	}
	review := false
	planned := map[string][]int{}
	for i, a := range actions {
		if a.State != "review" {
			k := a.DestinationID + "\x00" + strings.ToLower(a.TargetName)
			planned[k] = append(planned[k], i)
		}
	}
	for _, indices := range planned {
		if len(indices) < 2 {
			continue
		}
		for _, i := range indices {
			if actions[i].State == "done" {
				continue
			}
			actions[i].State = "review"
			actions[i].Error = "多個檔案產生相同名稱，保留原名待處理"
			if err := w.DB.SaveAction(ctx, actions[i]); err != nil {
				return err
			}
		}
	}
	for _, a := range actions {
		if a.State == "done" {
			continue
		}
		if a.State == "review" {
			review = true
			continue
		}
		current, e := api.Get(ctx, a.FileID)
		if e != nil {
			return e
		}
		if current.Phase != "" && !strings.Contains(strings.ToLower(current.Phase), "complete") {
			return &pikpak.Error{Kind: "transient", Message: "雲端檔案尚未完成，稍後整理"}
		}
		targets, e := pikpak.ListAll(ctx, api, a.DestinationID)
		if e != nil {
			return e
		}
		collision := false
		for _, target := range targets {
			if target.ID != a.FileID && strings.EqualFold(target.Name, a.TargetName) {
				collision = true
				break
			}
		}
		if collision {
			a.State = "review"
			a.Error = "目標已有同名檔案，保留原名；不覆寫"
			review = true
			if err := w.DB.SaveAction(ctx, a); err != nil {
				return err
			}
			continue
		}
		if current.Name != a.TargetName {
			if err := api.Rename(ctx, a.FileID, a.TargetName); err != nil {
				return err
			}
		}
		a.State = "renamed"
		if err := w.DB.SaveAction(ctx, a); err != nil {
			return err
		}
		if current.ParentID != a.DestinationID {
			if err := api.Move(ctx, a.FileID, a.DestinationID); err != nil {
				return err
			}
		}
		a.State = "done"
		a.Error = ""
		if err := w.DB.SaveAction(ctx, a); err != nil {
			return err
		}
	}
	j.Attempts = 0
	j.Error = ""
	j.State = "complete"
	if review {
		j.State = "needs_review"
		j.Error = "部分檔案無法解析或有命名衝突；原檔已保留，修改規則後可重新整理"
	}
	if err := w.DB.SaveJob(ctx, j); err != nil {
		return err
	}
	return w.DB.Event(ctx, j.ID, j.SubscriptionID, "info", map[bool]string{true: "雲端整理完成，部分檔案待處理", false: "離線下載與雲端整理完成"}[review])
}
