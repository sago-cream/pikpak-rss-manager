package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zeebo/bencode"

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

func TestSourcePagingProtectsCursorsAndDoesNotRepeatCachedNames(t *testing.T) {
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".torrent") {
			metadata, _ := bencode.EncodeBytes(map[string]any{"info": map[string]any{"name": strings.TrimPrefix(r.URL.Path, "/") + ".mkv", "length": 1, "piece length": 16384, "pieces": "01234567890123456789"}})
			_, _ = w.Write(metadata)
			return
		}
		fmt.Fprint(w, `<rss version="2.0"><channel><title>分頁</title>`)
		for i := 0; i < 5; i++ {
			fmt.Fprintf(w, `<item><guid>%d</guid><title>Episode %d</title><enclosure url="%d.torrent?private=torrent-sentinel" type="application/x-bittorrent"/></item>`, i, i, i)
		}
		fmt.Fprint(w, `</channel></rss>`)
	}))
	defer feedServer.Close()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cloud := testutil.NewCloud()
	manager := &folderManager{cloud}
	worker := worker.New(db, manager, feed.New(true))
	server, err := New(config.Config{AdminPassword: "x", AllowPrivateFeeds: true}, db, manager, worker, "test")
	if err != nil {
		t.Fatal(err)
	}
	webServer := httptest.NewServer(server.Handler())
	defer webServer.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	page, err := client.Get(webServer.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, page.Body)
	page.Body.Close()
	csrf := ""
	for _, cookie := range page.Cookies() {
		if cookie.Name == "pp_csrf" {
			csrf = cookie.Value
		}
	}
	post := func(path string, body any, protected bool) (int, []byte) {
		t.Helper()
		data, _ := json.Marshal(body)
		request, _ := http.NewRequest("POST", webServer.URL+path, bytes.NewReader(data))
		request.Header.Set("Content-Type", "application/json")
		if protected {
			request.Header.Set("X-CSRF-Token", csrf)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err = io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, data
	}
	if code, _ := post("/api/login", map[string]string{"password": "x"}, true); code != 200 {
		t.Fatal("fixture login failed")
	}
	sub := model.Subscription{Name: "分頁測試", RSSURL: feedServer.URL + "/feed?private=feed-sentinel", IntervalMinutes: 10}
	if err := db.SaveSubscription(context.Background(), &sub); err != nil {
		t.Fatal(err)
	}
	job := model.Job{ID: store.ID(), SubscriptionID: sub.ID, AccountID: cloud.AccountID, ResourceKey: "btih:cached", Title: "以前的發布", State: "complete"}
	if _, err := db.Enqueue(context.Background(), job, "cached"); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveAction(context.Background(), model.FileAction{JobID: job.ID, FileID: "cached-file", OriginalName: "cached-original.mkv", State: "done"}); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"url": sub.RSSURL, "subscription_id": sub.ID}
	code, data := post("/api/feeds/samples", body, true)
	var first feed.Samples
	if code != 200 || json.Unmarshal(data, &first) != nil || len(first.Items) != 4 || first.Items[0].Kind != "downloaded_file" || first.NextCursor == "" {
		t.Fatal("first sample page or cached name failed", code)
	}
	for _, private := range []string{"feed-sentinel", "torrent-sentinel", cloud.AccountID} {
		if strings.Contains(string(data), private) {
			t.Fatal("paging exposed private source metadata")
		}
	}
	body["cursor"] = first.NextCursor
	if code, _ := post("/api/feeds/samples", body, false); code != 403 {
		t.Fatal("continuation bypassed CSRF")
	}
	code, data = post("/api/feeds/samples", body, true)
	var next feed.Samples
	if code != 200 || json.Unmarshal(data, &next) != nil || len(next.Items) != 2 || next.NextCursor != "" {
		t.Fatal("continuation failed", code)
	}
	for i, sample := range next.Items {
		if sample.Kind != "torrent_file" || sample.Filename != fmt.Sprintf("%d.torrent.mkv", i+3) {
			t.Fatal("cached names repeated or torrent skipped", sample)
		}
	}
	body["cursor"] = "invalid"
	if code, _ := post("/api/feeds/samples", body, true); code != 400 {
		t.Fatal("malformed continuation accepted")
	}
	cloud.AccountID = "other-account"
	body["cursor"] = ""
	code, data = post("/api/feeds/samples", body, true)
	if code != 200 || json.Unmarshal(data, &next) != nil || len(next.Items) != 3 || next.Items[0].Kind != "torrent_file" {
		t.Fatal("old account's cached filenames reused")
	}
	stored, _ := db.Subscription(context.Background(), sub.ID)
	jobs, _ := db.Jobs(context.Background(), 10)
	if stored.Initialized || len(jobs) != 1 || cloud.Calls["submit"] != 0 || cloud.Calls["mkdir"] != 0 {
		t.Fatal("source paging changed baseline or cloud content")
	}
}
