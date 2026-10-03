package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestManualJobAuthenticationDeduplicationAndDestination(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	c := testutil.NewCloud()
	c.Files["folder"] = pikpak.File{ID: "folder", Name: "Fixture", Kind: "drive#folder"}
	c.Files["file"] = pikpak.File{ID: "file", Name: "file.txt", Kind: "drive#file"}
	m := &folderManager{c}
	w := worker.New(db, m, feed.New(false))
	s, err := initializedServer(t, "x", store.AppSettings{}, db, m, w)
	if err != nil {
		t.Fatal(err)
	}
	h := s.Handler()
	s.sessions["fixture-session"] = session{until: time.Now().Add(time.Hour)}
	csrf := strings.Repeat("c", 43)
	request := func(body any, auth, token bool) *httptest.ResponseRecorder {
		data, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "/api/jobs", bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		if auth {
			r.AddCookie(&http.Cookie{Name: "pp_session", Value: "fixture-session"})
		}
		r.AddCookie(&http.Cookie{Name: "pp_csrf", Value: csrf})
		if token {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		response := httptest.NewRecorder()
		h.ServeHTTP(response, r)
		return response
	}
	in := map[string]string{"name": "Manual fixture", "url": "magnet:?xt=urn:btih:0123456789ABCDEF0123456789ABCDEF01234567", "source_type": "torrent", "destination_id": "folder", "destination_account_ref": s.accountReference(c.AccountID)}
	if r := request(in, false, true); r.Code != 401 {
		t.Fatal("anonymous job accepted")
	}
	if r := request(in, true, false); r.Code != 403 {
		t.Fatal("job without CSRF accepted")
	}
	r := request(in, true, true)
	var job model.Job
	if r.Code != 201 || json.Unmarshal(r.Body.Bytes(), &job) != nil {
		t.Fatal("manual queue failed", r.Code, r.Body.String())
	}
	if strings.Contains(r.Body.String(), c.AccountID) || strings.Contains(r.Body.String(), "magnet:") {
		t.Fatal("private job data exposed")
	}
	stored, err := db.Job(context.Background(), job.ID)
	if err != nil || stored.SubscriptionID != 0 || stored.State != "queued" || stored.DestinationID != "folder" || stored.Destination != "Fixture" || stored.Rule.Renaming() || stored.ResourceKey != "btih:0123456789abcdef0123456789abcdef01234567" {
		t.Fatal("incorrect durable snapshot", err)
	}
	if c.Calls["submit"] != 0 || c.Calls["mkdir"] != 0 {
		t.Fatal("queue endpoint performs cloud mutations")
	}
	if r := request(in, true, true); r.Code != 409 {
		t.Fatal("duplicate source accepted")
	}
	// Feed jobs and manual jobs share the account/infohash uniqueness constraint.
	off := false
	sub := model.Subscription{Name: "Feed fixture", RSSURL: "https://example.test/feed", IntervalMinutes: 10, RenameEnabled: &off}
	if err := w.SaveSubscription(context.Background(), &sub); err != nil {
		t.Fatal(err)
	}
	feedJob := stored
	feedJob.ID = store.ID()
	feedJob.SubscriptionID = sub.ID
	if added, err := db.Enqueue(context.Background(), feedJob, "fixture"); err != nil || added {
		t.Fatal("cross-source deduplication failed", err)
	}
	if got, _ := db.Subscription(context.Background(), sub.ID); got.Initialized {
		t.Fatal("manual job changed baseline")
	}
	in["destination_account_ref"] = "stale"
	in["url"] = "magnet:?xt=urn:btih:1111111111111111111111111111111111111111"
	if r := request(in, true, true); r.Code != 400 {
		t.Fatal("stale destination accepted")
	}
	in["destination_account_ref"] = s.accountReference(c.AccountID)
	in["destination_id"] = "file"
	if r := request(in, true, true); r.Code != 400 {
		t.Fatal("file destination accepted")
	}
	in["destination_id"] = "folder"
	c.AccountID = "new-account"
	if r := request(in, true, true); r.Code != 400 {
		t.Fatal("previous account destination accepted")
	}
	in["destination_id"] = ""
	if r := request(in, true, true); r.Code != 400 {
		t.Fatal("stale root selection accepted")
	}
	in["destination_account_ref"] = ""
	in["destination"] = "Downloads"
	in["source_type"] = "url"
	in["url"] = "https://example.test/file.mp4?private=fixture"
	r = request(in, true, true)
	if r.Code != 201 {
		t.Fatal("direct URL rejected", r.Body.String())
	}
	if strings.Contains(r.Body.String(), "private=fixture") {
		t.Fatal("private source exposed")
	}
	for _, bad := range []string{"file:///private", "http://user:password@example.test/file", "", strings.Repeat("x", 8193)} {
		in["url"] = bad
		if r := request(in, true, true); r.Code != 400 {
			t.Fatal("invalid URL accepted")
		}
	}
	in["url"] = "https://example.test/file"
	in["source_type"] = "unknown"
	if r := request(in, true, true); r.Code != 400 {
		t.Fatal("unknown source type accepted")
	}
	in["source_type"] = "url"
	in["destination"] = "../unsafe"
	if r := request(in, true, true); r.Code != 400 {
		t.Fatal("invalid destination accepted")
	}
	jobs, _ := db.Jobs(context.Background(), 200)
	if len(jobs) != 2 {
		t.Fatal("rejected requests created jobs", len(jobs))
	}
}

func TestManualTorrentMetadataPrivateNetworkPolicy(t *testing.T) {
	metadata, _ := bencode.EncodeBytes(map[string]any{"info": map[string]any{"name": "fixture.txt", "length": 1, "piece length": 16384, "pieces": "01234567890123456789"}})
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(metadata) }))
	defer source.Close()
	for _, allowed := range []bool{false, true} {
		t.Run(map[bool]string{false: "blocked", true: "allowed"}[allowed], func(t *testing.T) {
			db, err := store.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			m := &folderManager{testutil.NewCloud()}
			w := worker.New(db, m, feed.New(allowed))
			s, err := initializedServer(t, "x", store.AppSettings{AllowPrivateFeeds: allowed}, db, m, w)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("POST", "/api/jobs", strings.NewReader(`{"name":"Torrent fixture","source_type":"torrent","url":"`+source.URL+`"}`))
			r.Header.Set("Content-Type", "application/json")
			result := httptest.NewRecorder()
			s.createJob(result, r)
			if allowed && result.Code != 201 || !allowed && result.Code != 400 {
				t.Fatal("private metadata policy", result.Code)
			}
		})
	}
}
