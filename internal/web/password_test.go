package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/config"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

type passwordClient struct {
	t       *testing.T
	handler http.Handler
	csrf    *http.Cookie
	session *http.Cookie
}

func (c *passwordClient) request(path string, value any, token, origin string) *httptest.ResponseRecorder {
	c.t.Helper()
	var b []byte
	if raw, ok := value.(json.RawMessage); ok {
		b = raw
	} else {
		var err error
		b, err = json.Marshal(value)
		if err != nil {
			c.t.Fatal(err)
		}
	}
	r := httptest.NewRequest("POST", path, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", token)
	r.Header.Set("Origin", origin)
	r.AddCookie(c.csrf)
	if c.session != nil {
		r.AddCookie(c.session)
	}
	w := httptest.NewRecorder()
	c.handler.ServeHTTP(w, r)
	return w
}

func newPasswordClient(t *testing.T, handler http.Handler, password string) *passwordClient {
	t.Helper()
	c := &passwordClient{t: t, handler: handler}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/session", nil))
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "pp_csrf" {
			c.csrf = cookie
		}
	}
	if c.csrf == nil {
		t.Fatal("missing CSRF")
	}
	w = c.request("/api/login", map[string]string{"password": password}, c.csrf.Value, "http://example.com")
	if w.Code != 200 {
		t.Fatal("fixture login failed", w.Code)
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "pp_session" {
			c.session = cookie
		}
	}
	return c
}

func (c *passwordClient) authenticated() bool {
	r := httptest.NewRequest("GET", "/api/session", nil)
	r.AddCookie(c.session)
	w := httptest.NewRecorder()
	c.handler.ServeHTTP(w, r)
	var state struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		c.t.Fatal(err)
	}
	return state.Authenticated
}

func TestChangePasswordValidationRevocationAndRestart(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	settings := store.AppSettings{AllowPrivateFeeds: true}
	s, err := initializedServer(t, "initial-fixture-password", settings, db, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	h := s.Handler()
	c := newPasswordClient(t, h, "initial-fixture-password")
	other := newPasswordClient(t, h, "initial-fixture-password")
	previous, _, _ := db.Administrator(context.Background())
	valid := map[string]string{"current_password": "initial-fixture-password", "new_password": "x", "confirm_password": "x"}
	for _, tc := range []struct {
		name   string
		body   any
		token  string
		origin string
		code   int
	}{
		{"missing CSRF", valid, "", "http://example.com", 403},
		{"cross origin", valid, c.csrf.Value, "https://evil.test", 403},
		{"empty current", map[string]string{"new_password": "x", "confirm_password": "x"}, c.csrf.Value, "http://example.com", 400},
		{"empty new", map[string]string{"current_password": "initial-fixture-password"}, c.csrf.Value, "http://example.com", 400},
		{"mismatch", map[string]string{"current_password": "initial-fixture-password", "new_password": "x", "confirm_password": "y"}, c.csrf.Value, "http://example.com", 400},
		{"wrong current", map[string]string{"current_password": "wrong", "new_password": "x", "confirm_password": "x"}, c.csrf.Value, "http://example.com", 400},
		{"unknown field", json.RawMessage(`{"extra":true}`), c.csrf.Value, "http://example.com", 400},
		{"multiple objects", json.RawMessage(`{} {}`), c.csrf.Value, "http://example.com", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := c.request("/api/settings/password", tc.body, tc.token, tc.origin)
			if w.Code != tc.code {
				t.Fatal("unexpected response", w.Code, w.Body.String())
			}
			hash, got, err := db.Administrator(context.Background())
			if err != nil || hash != previous || got != settings || !c.authenticated() || !other.authenticated() {
				t.Fatal("failed change modified credentials, settings or sessions", err)
			}
		})
	}
	anonymous := *c
	anonymous.session = nil
	if w := anonymous.request("/api/settings/password", valid, c.csrf.Value, "http://example.com"); w.Code != 401 {
		t.Fatal("anonymous change accepted")
	}
	w := c.request("/api/settings/password", valid, c.csrf.Value, "http://example.com")
	if w.Code != 200 || strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "$argon2id$") {
		t.Fatal("password change failed or leaked credentials", w.Code)
	}
	hash, got, err := db.Administrator(context.Background())
	if err != nil || hash == previous || !verifyPassword(hash, "x") || verifyPassword(hash, "initial-fixture-password") || got != settings {
		t.Fatal("verifier or settings incorrect", err)
	}
	if c.authenticated() || other.authenticated() {
		t.Fatal("password change did not revoke all sessions")
	}
	var revoked, rotated bool
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "pp_session" {
			revoked = cookie.MaxAge < 0 && cookie.Value == "" && cookie.HttpOnly && cookie.SameSite == http.SameSiteStrictMode
		}
		if cookie.Name == "pp_csrf" {
			rotated = cookie.Value != c.csrf.Value
		}
	}
	if !revoked || !rotated {
		t.Fatal("session cookie not expired or CSRF not rotated")
	}
	if w := c.request("/api/settings/password", valid, c.csrf.Value, "http://example.com"); w.Code != 401 {
		t.Fatal("revoked session changed password")
	}
	if w := c.request("/api/login", map[string]string{"password": "initial-fixture-password"}, c.csrf.Value, "http://example.com"); w.Code != 401 {
		t.Fatal("old password still accepted")
	}
	// Reopening the SQLite store must retain only the new verifier.
	db.Close()
	db, err = store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s, err = New(config.Config{}, db, nil, nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	c = newPasswordClient(t, s.Handler(), "x")
	// No length/character rules or whitespace trimming after setup either.
	current := "x"
	for _, next := range []string{"密碼🔑", strings.Repeat("long-fixture-password-", 20), " leading and trailing spaces "} {
		w := c.request("/api/settings/password", map[string]string{"current_password": current, "new_password": next, "confirm_password": next}, c.csrf.Value, "http://example.com")
		if w.Code != 200 {
			t.Fatal("password policy changed", w.Code)
		}
		c = newPasswordClient(t, s.Handler(), next)
		current = next
	}
	db.Close()
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if bytes.Contains(b, []byte("initial-fixture-password")) || bytes.Contains(b, []byte(current)) {
				t.Fatal("plaintext password persisted")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestChangePasswordFailureAndThrottling(t *testing.T) {
	s, err := initializedServer(t, "x", store.AppSettings{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := newPasswordClient(t, s.Handler(), "x")
	valid := map[string]string{"current_password": "x", "new_password": "new-fixture-password", "confirm_password": "new-fixture-password"}
	s.authGate <- struct{}{}
	if w := c.request("/api/settings/password", valid, c.csrf.Value, "http://example.com"); w.Code != 429 {
		t.Fatal("parallel Argon2 operation accepted")
	}
	<-s.authGate
	previous := s.hash
	s.DB.Close()
	if w := c.request("/api/settings/password", valid, c.csrf.Value, "http://example.com"); w.Code != 500 || strings.Contains(w.Body.String(), "new-fixture-password") {
		t.Fatal("storage failure not handled safely", w.Code)
	}
	if s.hash != previous || !c.authenticated() {
		t.Fatal("storage failure changed in-memory password or revoked session")
	}
	newPasswordClient(t, s.Handler(), "x")
	wrong := map[string]string{"current_password": "wrong", "new_password": "x", "confirm_password": "x"}
	for i := 0; i < 8; i++ {
		if w := c.request("/api/settings/password", wrong, c.csrf.Value, "http://example.com"); w.Code != 400 {
			t.Fatal("wrong-password attempt did not fail", i, w.Code)
		}
	}
	if w := c.request("/api/settings/password", valid, c.csrf.Value, "http://example.com"); w.Code != 429 {
		t.Fatal("password guesses not throttled")
	}
}
