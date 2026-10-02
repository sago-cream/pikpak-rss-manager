package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/config"
	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/rename"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/worker"
)

func TestAuthenticatedAPIAndSecretBoundaries(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m := pikpak.NewManager(db, "", "environment")
	w := worker.New(db, m, feed.New(false))
	s, err := New(config.Config{AdminPassword: "unit-test-password-only"}, db, m, w, "test")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	request := func(method, path, body, csrf, origin string) (int, string, http.Header) {
		t.Helper()
		req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b), res.Header
	}
	code, _, headers := request("GET", "/healthz", "", "", "")
	if code != 200 || !strings.Contains(headers.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatal("health/security headers")
	}
	code, _, _ = request("GET", "/api/subscriptions", "", "", "")
	if code != 401 {
		t.Fatal("anonymous read allowed")
	}
	_, session, _ := request("GET", "/api/session", "", "", "")
	var info struct {
		CSRF string `json:"csrf"`
	}
	_ = json.Unmarshal([]byte(session), &info)
	code, _, _ = request("POST", "/api/login", `{"password":"unit-test-password-only"}`, "", "")
	if code != 403 {
		t.Fatal("login CSRF missing")
	}
	code, _, _ = request("POST", "/api/login", `{"password":"unit-test-password-only"}`, info.CSRF, "https://evil.test")
	if code != 403 {
		t.Fatal("cross-origin login allowed")
	}
	code, _, headers = request("POST", "/api/login", `{"password":"unit-test-password-only"}`, info.CSRF, srv.URL)
	if code != 200 {
		t.Fatal("login failed")
	}
	if !strings.Contains(headers.Get("Set-Cookie"), "HttpOnly") || !strings.Contains(headers.Get("Set-Cookie"), "SameSite=Strict") {
		t.Fatal("session cookie protections missing")
	}
	sub := model.Subscription{Name: "葬送的芙莉蓮", RSSURL: "https://rss.test/feed?private=test-rss-key", Destination: "Anime/芙莉蓮", IntervalMinutes: 10, Enabled: false, Season: 1, Regex: rename.DefaultRegex, Template: rename.DefaultTemplate}
	b, _ := json.Marshal(sub)
	code, _, _ = request("POST", "/api/subscriptions", string(b), "", "")
	if code != 403 {
		t.Fatal("mutation CSRF missing")
	}
	code, body, _ := request("POST", "/api/subscriptions", string(b), info.CSRF, srv.URL)
	if code != 200 {
		t.Fatal("save failed", body)
	}
	_ = json.Unmarshal([]byte(body), &sub)
	if sub.ID == 0 {
		t.Fatal("created subscription missing id")
	}
	preview, _ := json.Marshal(map[string]any{"rule": sub.Rule(), "title": "[字幕組] S01E03", "filename": "original.mp4"})
	code, body, _ = request("POST", "/api/rules/preview", string(preview), info.CSRF, srv.URL)
	if code != 200 || !strings.Contains(body, "S01E03.mp4") {
		t.Fatal("preview API mismatch", body)
	}
	code, _, _ = request("POST", "/api/rules/preview", string(preview)+" {}", info.CSRF, srv.URL)
	if code != 400 {
		t.Fatal("trailing JSON accepted")
	}
	secret := "unique-provider-secret-for-this-test"
	_ = db.SetSetting(context.Background(), "pikpak_token", secret)
	j := model.Job{ID: store.ID(), SubscriptionID: sub.ID, AccountID: "private-account-id", ResourceKey: "btih:one", ResourceURL: "magnet:?private=test-tracker-key", State: "queued"}
	if _, err := db.Enqueue(context.Background(), j, "one"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/jobs", "/api/jobs/" + j.ID, "/api/settings/pikpak", "/api/events"} {
		code, body, _ = request("GET", path, "", "", "")
		if code != 200 {
			t.Fatal(path, code)
		}
		for _, private := range []string{secret, j.AccountID, j.ResourceURL, "test-tracker-key"} {
			if bytes.Contains([]byte(body), []byte(private)) {
				t.Fatal("private field escaped through", path)
			}
		}
	}
	code, _, _ = request("POST", "/api/logout", `{}`, info.CSRF, srv.URL)
	if code != 200 {
		t.Fatal("logout failed")
	}
	code, _, _ = request("GET", "/api/jobs", "", "", "")
	if code != 401 {
		t.Fatal("logout left usable session")
	}
}
