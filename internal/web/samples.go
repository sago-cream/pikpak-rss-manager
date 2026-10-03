package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/feed"
)

func (s *Server) sourceSamples(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL            string `json:"url"`
		SubscriptionID int64  `json:"subscription_id"`
		Cursor         string `json:"cursor"`
		All            bool   `json:"all"`
	}
	if err := decode(w, r, &in); err != nil {
		failure(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	in.URL = strings.TrimSpace(in.URL)
	client := feed.New(s.appSettings().AllowPrivateFeeds)
	defer client.HTTP.CloseIdleConnections()
	var out feed.Samples
	var err error
	if in.All {
		if in.Cursor != "" {
			failure(w, errors.New("一次讀取全部種子檔名不使用續讀位置"))
			return
		}
		out, err = client.SamplesAll(ctx, in.URL)
	} else {
		out, err = client.SamplesPage(ctx, in.URL, in.Cursor)
	}
	if err != nil {
		failure(w, err)
		return
	}
	// Persisted action plans provide actual original names for a saved
	// subscription without making another cloud request or download task.
	if in.Cursor == "" && in.SubscriptionID > 0 && s.DB != nil && s.Manager != nil {
		sub, err := s.DB.Subscription(ctx, in.SubscriptionID)
		_, account := s.Manager.Snapshot()
		if err == nil && sub.RSSURL == strings.TrimSpace(in.URL) && account != "" {
			jobs, _ := s.DB.Jobs(ctx, 200)
			known := []feed.Sample{}
			for _, job := range jobs {
				if job.SubscriptionID != sub.ID || job.AccountID != account || len(known) >= 10 {
					continue
				}
				actions, _ := s.DB.Actions(ctx, job.ID)
				for _, action := range actions {
					if action.OriginalName != "" && len(known) < 10 {
						known = append(known, feed.Sample{Title: job.Title, Filename: action.OriginalName, Kind: "downloaded_file"})
					}
				}
			}
			out.Items = append(known, out.Items...)
		}
	}
	JSON(w, 200, out)
}
