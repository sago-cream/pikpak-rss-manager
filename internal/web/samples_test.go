package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/config"
	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/testutil"
	"github.com/wade00754/pikpak-rss-manager/internal/worker"
)

func TestSourcePreviewIsAuthenticatedAndDoesNotQueueDownloads(t *testing.T) {
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<rss version="2.0"><channel><title>範例</title><item><title>RSS 標題 / Example [01]</title><link>magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567</link></item></channel></rss>`)
	}))
	defer feedServer.Close()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	c := testutil.NewCloud()
	m := &folderManager{c}
	w := worker.New(db, m, feed.New(true))
	s, err := New(config.Config{AdminPassword: "x", AllowPrivateFeeds: true}, db, m, w, "test")
	if err != nil {
		t.Fatal(err)
	}
	handler := s.Handler()
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest("GET", "/", nil))
	var cookies []*http.Cookie
	csrf := ""
	for _, cookie := range page.Result().Cookies() {
		cookies = append(cookies, cookie)
		if cookie.Name == "pp_csrf" {
			csrf = cookie.Value
		}
	}
	request := func(path string, body any, authenticated, protected bool) *httptest.ResponseRecorder {
		t.Helper()
		data, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", path, bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		if protected {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		for _, cookie := range cookies {
			if authenticated || cookie.Name == "pp_csrf" {
				r.AddCookie(cookie)
			}
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, r)
		cookies = append(cookies, response.Result().Cookies()...)
		return response
	}
	if r := request("/api/feeds/samples", map[string]string{"url": feedServer.URL}, false, true); r.Code != 401 {
		t.Fatal("anonymous source request allowed")
	}
	if r := request("/api/login", map[string]string{"password": "x"}, false, true); r.Code != 200 {
		t.Fatal("login failed")
	}
	sub := model.Subscription{Name: "作品", RSSURL: feedServer.URL + "?private=rss-sentinel", IntervalMinutes: 10}
	if err := db.SaveSubscription(context.Background(), &sub); err != nil {
		t.Fatal(err)
	}
	job := model.Job{ID: store.ID(), SubscriptionID: sub.ID, AccountID: c.AccountID, ResourceKey: "btih:test", ResourceURL: "magnet:?private=tracker-sentinel", Title: "以前的發布", State: "complete"}
	if _, err := db.Enqueue(context.Background(), job, "previous"); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveAction(context.Background(), model.FileAction{JobID: job.ID, FileID: "test-file", OriginalName: "真實原始檔名 [01].mkv", State: "done"}); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"url": sub.RSSURL, "subscription_id": sub.ID}
	if r := request("/api/feeds/samples", body, true, false); r.Code != 403 {
		t.Fatal("sample request bypassed CSRF")
	}
	r := request("/api/feeds/samples", body, true, true)
	var out feed.Samples
	if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &out) != nil || len(out.Items) != 2 || out.Items[0].Kind != "downloaded_file" || out.Items[1].Kind != "rss_title" {
		t.Fatal("source filename provenance failed", r.Code, r.Body.String())
	}
	for _, value := range []string{"rss-sentinel", "tracker-sentinel", c.AccountID} {
		if strings.Contains(r.Body.String(), value) {
			t.Fatal("private metadata escaped in source preview")
		}
	}
	stored, _ := db.Subscription(context.Background(), sub.ID)
	jobs, _ := db.Jobs(context.Background(), 10)
	if stored.Initialized || len(jobs) != 1 || c.Calls["submit"] != 0 || c.Calls["mkdir"] != 0 {
		t.Fatal("preview changed subscription baseline or cloud downloads")
	}
	c.AccountID = "different-account"
	r = request("/api/feeds/samples", body, true, true)
	if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &out) != nil || len(out.Items) != 1 || out.Items[0].Kind != "rss_title" {
		t.Fatal("old account's filenames reused as current samples")
	}
}
