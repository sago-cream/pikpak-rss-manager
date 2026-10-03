package web

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/rename"
)

func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name          string `json:"name"`
		URL           string `json:"url"`
		SourceType    string `json:"source_type"`
		Destination   string `json:"destination"`
		DestinationID string `json:"destination_id"`
		AccountRef    string `json:"destination_account_ref"`
	}
	if err := decode(w, r, &in); err != nil {
		failure(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	off := false
	sub := model.Subscription{Name: strings.TrimSpace(in.Name), Destination: in.Destination, DestinationID: in.DestinationID, DestinationAccountRef: in.AccountRef, RenameEnabled: &off, RenameMode: "replace"}
	if err := rename.Validate(sub.Rule()); err != nil {
		failure(w, err)
		return
	}
	if sub.DestinationID == "" {
		destination, err := rename.Destination(sub.Destination)
		if err != nil {
			failure(w, err)
			return
		}
		sub.Destination = destination
	}
	_, originalAccount, err := s.directoryAPI()
	if err != nil {
		failure(w, err)
		return
	}
	source := strings.TrimSpace(in.URL)
	if len(source) == 0 || len(source) > 8192 {
		failure(w, errors.New("下載連結必須為 1–8192 位元組"))
		return
	}
	var resource feed.Resource
	switch in.SourceType {
	case "torrent":
		// Fetch metadata only, with the current private-network policy and limits.
		resource, err = feed.New(s.appSettings().AllowPrivateFeeds).Resolve(ctx, source)
	case "url":
		err = feed.ValidateURL(source)
		if err == nil {
			hash := sha256.Sum256([]byte(source))
			resource = feed.Resource{Key: "url:" + hex.EncodeToString(hash[:]), URL: source}
		}
	default:
		err = errors.New("請選擇下載連結類型")
	}
	if err != nil {
		failure(w, err)
		return
	}
	s.Worker.Gate.Lock()
	defer s.Worker.Gate.Unlock()
	_, account, err := s.directoryAPI()
	if err != nil {
		failure(w, err)
		return
	}
	if account != originalAccount {
		failure(w, errors.New("PikPak 帳號已更換，請重新建立任務"))
		return
	}
	if in.AccountRef != "" && !hmac.Equal([]byte(in.AccountRef), []byte(s.accountReference(account))) {
		failure(w, errors.New("PikPak 帳號或連線已更新，請重新選取目標資料夾"))
		return
	}
	if sub.DestinationID != "" {
		if err := s.resolveDestinationLocked(ctx, &sub); err != nil {
			failure(w, err)
			return
		}
	}
	job, added, err := s.Worker.EnqueueManual(ctx, account, resource, sub)
	if err != nil {
		failure(w, errors.New("無法建立任務"))
		return
	}
	if !added {
		JSON(w, http.StatusConflict, map[string]string{"error": "此來源已存在任務"})
		return
	}
	JSON(w, http.StatusCreated, job)
}
