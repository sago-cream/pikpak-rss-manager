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

func initializedServer(t *testing.T, password string, settings store.AppSettings, db *store.Store, m CloudManager, w *worker.Worker) (*Server, error) {
	t.Helper()
	if db == nil {
		var err error
		db, err = store.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
	}
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if created, err := db.InitializeAdministrator(context.Background(), hash, settings); err != nil || !created {
		t.Fatal("fixture setup failed", err)
	}
	return New(config.Config{}, db, m, w, "test")
}

func TestLoginWithoutPasswordLengthOrCharacterRules(t *testing.T) {
	for name, password := range map[string]string{
		"one character": "x",
		"unicode":       "密碼🔑",
		"long":          strings.Repeat("long-password-", 20),
		"spaces":        " leading and trailing spaces ",
	} {
		t.Run(name, func(t *testing.T) {
			s, err := initializedServer(t, password, store.AppSettings{}, nil, nil, nil)
			if err != nil {
				t.Fatal("password rejected at startup", err)
			}
			handler := s.Handler()
			page := httptest.NewRecorder()
			handler.ServeHTTP(page, httptest.NewRequest("GET", "/", nil))
			if page.Code != http.StatusOK || strings.Contains(page.Body.String(), "minlength=") || strings.Contains(page.Body.String(), "maxlength=") {
				t.Fatal("login form still restricts password length")
			}
			var csrf *http.Cookie
			for _, c := range page.Result().Cookies() {
				if c.Name == "pp_csrf" {
					csrf = c
				}
			}
			if csrf == nil {
				t.Fatal("missing CSRF cookie")
			}
			login := func(candidate string) *httptest.ResponseRecorder {
				t.Helper()
				body, _ := json.Marshal(map[string]string{"password": candidate})
				req := httptest.NewRequest("POST", "/api/login", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-CSRF-Token", csrf.Value)
				req.AddCookie(csrf)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, req)
				return response
			}
			// The long case differs only beyond byte 72: this must not be truncated.
			if response := login(password + "!"); response.Code != http.StatusUnauthorized {
				t.Fatal("incorrect password was accepted")
			}
			response := login(password)
			if response.Code != http.StatusOK {
				t.Fatal("correct password could not log in")
			}
			req := httptest.NewRequest("GET", "/api/session", nil)
			for _, c := range response.Result().Cookies() {
				req.AddCookie(c)
			}
			session := httptest.NewRecorder()
			handler.ServeHTTP(session, req)
			var state struct {
				Authenticated bool `json:"authenticated"`
			}
			if err := json.Unmarshal(session.Body.Bytes(), &state); err != nil || !state.Authenticated {
				t.Fatal("successful login did not create a usable session")
			}
		})
	}
}

func TestAuthenticatedAPIAndSecretBoundaries(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m := pikpak.NewManager(db)
	w := worker.New(db, m, feed.New(false))
	s, err := initializedServer(t, "unit-test-password-only", store.AppSettings{}, db, m, w)
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
