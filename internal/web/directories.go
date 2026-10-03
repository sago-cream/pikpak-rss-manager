package web

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
)

// An opaque account reference prevents a browser's stale selection from being
// reused after switching accounts. It never exposes the PikPak account ID.
func (s *Server) accountReference(account string) string {
	mac := hmac.New(sha256.New, s.accountKey)
	_, _ = mac.Write([]byte("folder-account\x00" + account))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) directoryReference(sub *model.Subscription) {
	sub.DestinationAccountRef = ""
	if sub.DestinationID != "" && sub.DestinationAccountID != "" {
		sub.DestinationAccountRef = s.accountReference(sub.DestinationAccountID)
	}
}

func (s *Server) directoryAPI() (pikpak.API, string, error) {
	if s.Manager == nil {
		return nil, "", errors.New("請先綁定 PikPak PAT")
	}
	api, account := s.Manager.Snapshot()
	if api == nil || account == "" {
		return nil, "", errors.New("請先綁定 PikPak，或解除授權／配額暫停")
	}
	return api, account, nil
}

func (s *Server) browseDirectories(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	api, account, err := s.directoryAPI()
	if err != nil {
		failure(w, err)
		return
	}
	listing, err := pikpak.BrowseDirectories(ctx, api, r.URL.Query().Get("parent_id"), r.URL.Query().Get("token"))
	if err != nil {
		failure(w, err)
		return
	}
	listing.AccountRef = s.accountReference(account)
	JSON(w, 200, listing)
}

func (s *Server) addDirectory(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ParentID   string `json:"parent_id"`
		Name       string `json:"name"`
		AccountRef string `json:"account_ref"`
	}
	if err := decode(w, r, &in); err != nil {
		failure(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	s.Worker.Gate.Lock()
	defer s.Worker.Gate.Unlock()
	if err := ctx.Err(); err != nil {
		failure(w, pikpak.Classify(err))
		return
	}
	api, account, err := s.directoryAPI()
	if err != nil {
		failure(w, err)
		return
	}
	if !hmac.Equal([]byte(in.AccountRef), []byte(s.accountReference(account))) {
		failure(w, errors.New("PikPak 帳號或連線已更新，請重新開啟資料夾選擇器"))
		return
	}
	folder, err := pikpak.AddDirectory(ctx, api, in.ParentID, in.Name)
	if err != nil {
		failure(w, err)
		return
	}
	JSON(w, 201, map[string]any{"folder": folder, "account_ref": in.AccountRef})
}

func (s *Server) resolveDestination(ctx context.Context, sub *model.Subscription) error {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	s.Worker.Gate.Lock()
	defer s.Worker.Gate.Unlock()
	return s.resolveDestinationLocked(ctx, sub)
}

// The caller holds Worker.Gate through validation and durable queue insertion.
func (s *Server) resolveDestinationLocked(ctx context.Context, sub *model.Subscription) error {
	if err := ctx.Err(); err != nil {
		return pikpak.Classify(err)
	}
	api, account, err := s.directoryAPI()
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(sub.DestinationAccountRef), []byte(s.accountReference(account))) {
		return errors.New("PikPak 帳號或連線已更新，請重新選取目標資料夾")
	}
	crumbs, err := pikpak.DirectoryPath(ctx, api, sub.DestinationID)
	if err != nil {
		return err
	}
	sub.Destination = crumbs[len(crumbs)-1].Path
	sub.DestinationAccountID = account
	return nil
}
