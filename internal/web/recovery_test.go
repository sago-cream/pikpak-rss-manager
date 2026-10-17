package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

func TestPasswordRecoveryCSRFRevocationAndRestart(t *testing.T) {
	ctx, dir := context.Background(), t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	settings := store.AppSettings{AllowPrivateFeeds: true}
	s, err := initializedServer(t, "original-password", settings, db, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetSetting(ctx, "pikpak_token", "fixture-pat"); err != nil {
		t.Fatal(err)
	}
	c := newPasswordClient(t, s.Handler(), "original-password")
	other := newPasswordClient(t, s.Handler(), "original-password")
	anonymous := *c
	anonymous.session = nil
	page := httptest.NewRecorder()
	s.Handler().ServeHTTP(page, httptest.NewRequest("GET", "/forgot-password", nil))
	if page.Code != 200 || !strings.Contains(page.Body.String(), `data-page="recovery"`) {
		t.Fatal("anonymous recovery page missing")
	}
	code, err := db.IssuePasswordRecovery(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]string{"code": code, "new_password": "x", "confirm_password": "x"}
	for _, tc := range []struct {
		name, token, origin string
	}{
		{"missing CSRF", "", "http://example.com"},
		{"cross origin", c.csrf.Value, "https://evil.test"},
	} {
		if w := anonymous.request("/api/password/recovery", body, tc.token, tc.origin); w.Code != 403 {
			t.Fatal(tc.name, "accepted", w.Code)
		}
	}
	body["confirm_password"] = "different"
	if w := anonymous.request("/api/password/recovery", body, c.csrf.Value, "http://example.com"); w.Code != 400 || !c.authenticated() {
		t.Fatal("mismatch changed password or revoked session")
	}
	body["confirm_password"] = "x"
	body["code"] = strings.Repeat("a", 43)
	if w := anonymous.request("/api/password/recovery", body, c.csrf.Value, "http://example.com"); w.Code != 400 || !c.authenticated() {
		t.Fatal("wrong recovery code accepted or changed session")
	}
	body["code"] = code
	w := anonymous.request("/api/password/recovery", body, c.csrf.Value, "http://example.com")
	if w.Code != 200 || strings.Contains(w.Body.String(), code) || strings.Contains(w.Body.String(), "password") {
		t.Fatal("recovery failed or leaked credentials", w.Code)
	}
	if c.authenticated() || other.authenticated() {
		t.Fatal("recovery did not revoke all sessions")
	}
	if w := anonymous.request("/api/password/recovery", body, c.csrf.Value, "http://example.com"); w.Code != 400 {
		t.Fatal("recovery code could be replayed")
	}
	hash, got, err := db.Administrator(ctx)
	if err != nil || got != settings || !verifyPassword(hash, "x") || verifyPassword(hash, "original-password") {
		t.Fatal("recovery verifier or site settings incorrect", err)
	}
	if token, err := db.Setting(ctx, "pikpak_token"); err != nil || token != "fixture-pat" {
		t.Fatal("recovery lost cloud credentials", err)
	}
	reopened, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted, err := New(reopened, nil, nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	_ = newPasswordClient(t, restarted.Handler(), "x")
}

func TestPasswordRecoveryExpiredAndThrottled(t *testing.T) {
	s, err := initializedServer(t, "fixture-password", store.AppSettings{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := newPasswordClient(t, s.Handler(), "fixture-password")
	c.session = nil
	code, err := s.DB.IssuePasswordRecovery(context.Background(), time.Now().Add(-store.PasswordRecoveryTTL))
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]string{"code": code, "new_password": "new", "confirm_password": "new"}
	for range 8 {
		if w := c.request("/api/password/recovery", body, c.csrf.Value, "http://example.com"); w.Code != 400 {
			t.Fatal("expired code not rejected", w.Code)
		}
	}
	if w := c.request("/api/password/recovery", body, c.csrf.Value, "http://example.com"); w.Code != http.StatusTooManyRequests {
		t.Fatal("recovery attempts not throttled", w.Code)
	}
	if !verifyPassword(s.hash, "fixture-password") {
		t.Fatal("expired code changed verifier")
	}
}
