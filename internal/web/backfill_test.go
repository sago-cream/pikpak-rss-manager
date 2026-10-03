package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/testutil"
	"github.com/wade00754/pikpak-rss-manager/internal/worker"
	"github.com/zeebo/bencode"
)

func TestBackfillSelectionAuthenticationReplayAndSnapshot(t *testing.T) {
	metadata, _ := bencode.EncodeBytes(map[string]any{"info": map[string]any{"name": "Fixture [01].mkv", "length": 1, "piece length": 16384, "pieces": "01234567890123456789"}})
	var feedReads, torrentReads atomic.Int32
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed" {
			feedReads.Add(1)
			fmt.Fprint(w, `<rss version="2.0"><channel><title>Fixture</title><item><guid>one</guid><title>第一筆</title><enclosure url="/one.torrent" type="application/x-bittorrent"/></item><item><guid>two</guid><title>第二筆</title><enclosure url="/one.torrent" type="application/x-bittorrent"/></item></channel></rss>`)
		} else {
			torrentReads.Add(1)
			w.Write(metadata)
		}
	}))
	defer source.Close()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cloud := testutil.NewCloud()
	cloud.Files["dest"] = pikpak.File{ID: "dest", Name: "Fixture", Kind: "drive#folder"}
	m := &folderManager{cloud}
	w := worker.New(db, m, feed.New(true))
	off := false
	sub := model.Subscription{Name: "Fixture", RSSURL: source.URL + "/feed", Destination: "Fixture", DestinationID: "dest", DestinationAccountID: cloud.AccountID, IntervalMinutes: 10, RenameEnabled: &off}
	if err := w.SaveSubscription(context.Background(), &sub); err != nil {
		t.Fatal(err)
	}
	s, err := initializedServer(t, "x", store.AppSettings{AllowPrivateFeeds: true}, db, m, w)
	if err != nil {
		t.Fatal(err)
	}
	s.sessions["fixture"] = session{until: time.Now().Add(time.Hour)}
	h := s.Handler()
	csrf := strings.Repeat("c", 43)
	request := func(path string, body any, auth, token bool) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", fmt.Sprintf("/api/subscriptions/%d/%s", sub.ID, path), bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		if auth {
			r.AddCookie(&http.Cookie{Name: "pp_session", Value: "fixture"})
		}
		r.AddCookie(&http.Cookie{Name: "pp_csrf", Value: csrf})
		if token {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		return out
	}
	for _, path := range []string{"backfill/preview", "backfill"} {
		if r := request(path, map[string]any{}, false, true); r.Code != 401 {
			t.Fatal("anonymous selection accepted")
		}
		if r := request(path, map[string]any{}, true, false); r.Code != 403 {
			t.Fatal("selection without CSRF accepted")
		}
	}
	preview := func() (string, []backfillChoice) {
		r := request("backfill/preview", map[string]any{}, true, true)
		var p struct {
			Token string           `json:"token"`
			Items []backfillChoice `json:"items"`
		}
		if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &p) != nil || len(p.Items) != 2 || len(p.Items[0].Filenames) != 1 || p.Items[0].Filenames[0] != "Fixture [01].mkv" {
			t.Fatal("preview failed", r.Code, r.Body.String())
		}
		if strings.Contains(r.Body.String(), source.URL) || strings.Contains(r.Body.String(), cloud.AccountID) {
			t.Fatal("preview leaked private sources")
		}
		return p.Token, p.Items
	}
	token, choices := preview()
	if feedReads.Load() != 1 || torrentReads.Load() != 1 {
		t.Fatal("metadata fetched more than once", feedReads.Load(), torrentReads.Load())
	}
	jobs, _ := db.Jobs(context.Background(), 10)
	got, _ := db.Subscription(context.Background(), sub.ID)
	if len(jobs) != 0 || got.Initialized || cloud.Calls["submit"] != 0 || cloud.Calls["mkdir"] != 0 {
		t.Fatal("preview changed baseline or cloud")
	}
	in := map[string]any{"token": token, "selected": []string{choices[0].ID}, "overwrite": false}
	if r := request("backfill", in, true, true); r.Code != 400 {
		t.Fatal("missing confirmation accepted")
	}
	in["overwrite"] = true
	in["selected"] = []string{"forged"}
	if r := request("backfill", in, true, true); r.Code != 400 {
		t.Fatal("forged selection accepted")
	}
	in["selected"] = []string{choices[0].ID}
	cloud.AccountID = "switched"
	if r := request("backfill", in, true, true); r.Code != 400 {
		t.Fatal("account switch accepted")
	}
	cloud.AccountID = "test-account"
	for range 2 {
		if r := request("backfill", in, true, true); r.Code != 200 {
			t.Fatal("confirm/replay failed", r.Body.String())
		}
	}
	jobs, _ = db.Jobs(context.Background(), 10)
	if len(jobs) != 1 || jobs[0].Title != choices[0].Title || !jobs[0].Overwrite || jobs[0].State != "queued" || jobs[0].DestinationID != "dest" {
		t.Fatal("selection/replay incorrect")
	}
	in["selected"] = []string{choices[1].ID}
	if r := request("backfill", in, true, true); r.Code != 400 {
		t.Fatal("changed replay accepted")
	}
	// A fresh explicit selection of the same resource creates a new task.
	token, choices = preview()
	in["token"] = token
	in["selected"] = []string{choices[0].ID}
	if r := request("backfill", in, true, true); r.Code != 200 {
		t.Fatal("redownload suppressed", r.Body.String())
	}
	jobs, _ = db.Jobs(context.Background(), 10)
	if len(jobs) != 2 {
		t.Fatal("repeat resource suppressed")
	}
	token, choices = preview()
	in["token"] = token
	in["selected"] = []string{choices[0].ID}
	sub.Name = "Changed"
	if err := w.SaveSubscription(context.Background(), &sub); err != nil {
		t.Fatal(err)
	}
	if r := request("backfill", in, true, true); r.Code != 400 {
		t.Fatal("stale subscription snapshot accepted")
	}
	s.backfills[token].until = time.Now().Add(-time.Second)
	if r := request("backfill", in, true, true); r.Code != 400 {
		t.Fatal("expired plan accepted")
	}
}
