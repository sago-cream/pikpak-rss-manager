package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/testutil"
	"github.com/wade00754/pikpak-rss-manager/internal/worker"
)

type folderManager struct{ cloud *testutil.Cloud }

func (m *folderManager) Snapshot() (pikpak.API, string) { return m.cloud, m.cloud.AccountID }
func (m *folderManager) Status() pikpak.Status {
	return pikpak.Status{Connected: true, Name: "測試帳號"}
}
func (m *folderManager) Pause(*pikpak.Error)                {}
func (m *folderManager) Reconnect(context.Context) error    { return nil }
func (m *folderManager) Bind(context.Context, string) error { return nil }

func TestFolderAPIAuthenticationSelectionAndAccountIsolation(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	c := testutil.NewCloud()
	c.Files["anime"] = pikpak.File{ID: "anime", Name: "Anime", Kind: "drive#folder"}
	c.Files["one"] = pikpak.File{ID: "one", Name: "作品", Kind: "drive#folder", ParentID: "anime"}
	c.Files["two"] = pikpak.File{ID: "two", Name: "作品", Kind: "drive#folder", ParentID: "anime"}
	c.Files["video"] = pikpak.File{ID: "video", Name: "private-video.mp4", Kind: "drive#file"}
	m := &folderManager{c}
	w := worker.New(db, m, feed.New(false))
	s, err := initializedServer(t, "x", store.AppSettings{}, db, m, w)
	if err != nil {
		t.Fatal(err)
	}
	handler := s.Handler()
	csrf := ""
	cookies := []*http.Cookie{}
	request := func(method, path string, body any, authenticated, withCSRF bool) *httptest.ResponseRecorder {
		t.Helper()
		var data []byte
		if body != nil {
			data, _ = json.Marshal(body)
		}
		r := httptest.NewRequest(method, path, bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		if withCSRF {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		for _, cookie := range cookies {
			if authenticated || cookie.Name == "pp_csrf" {
				r.AddCookie(cookie)
			}
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, r)
		for _, cookie := range response.Result().Cookies() {
			cookies = append(cookies, cookie)
			if cookie.Name == "pp_csrf" {
				csrf = cookie.Value
			}
		}
		return response
	}
	if r := request("GET", "/api/pikpak/folders", nil, false, false); r.Code != 401 {
		t.Fatal("anonymous folder listing allowed")
	}
	request("GET", "/", nil, false, false)
	if r := request("POST", "/api/login", map[string]string{"password": "x"}, false, true); r.Code != 200 {
		t.Fatal("login failed")
	}
	r := request("GET", "/api/pikpak/folders", nil, true, false)
	var listing pikpak.Directories
	if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &listing) != nil || len(listing.Folders) != 1 || listing.AccountRef == "" {
		t.Fatal("folder listing failed", r.Code)
	}
	if strings.Contains(r.Body.String(), c.AccountID) || strings.Contains(r.Body.String(), "private-video.mp4") {
		t.Fatal("unneeded private metadata exposed")
	}
	create := map[string]string{"parent_id": "two", "name": "新作品", "account_ref": listing.AccountRef}
	if r := request("POST", "/api/pikpak/folders", create, true, false); r.Code != 403 || c.Calls["mkdir"] != 0 {
		t.Fatal("folder mutation without CSRF")
	}
	r = request("POST", "/api/pikpak/folders", create, true, true)
	var made struct {
		Folder pikpak.Directory `json:"folder"`
	}
	if r.Code != 201 || json.Unmarshal(r.Body.Bytes(), &made) != nil || made.Folder.Path != "Anime/作品/新作品" || c.Files[made.Folder.ID].ParentID != "two" {
		t.Fatal("folder creation failed", r.Code)
	}
	if r := request("POST", "/api/pikpak/folders", create, true, true); r.Code != 400 || c.Calls["mkdir"] != 1 {
		t.Fatal("duplicate folder was created")
	}
	off := false
	sub := model.Subscription{Name: "測試作品", RSSURL: "https://rss.test", Destination: "spoofed", DestinationID: "two", DestinationAccountRef: listing.AccountRef, IntervalMinutes: 10, RenameEnabled: off}
	r = request("POST", "/api/subscriptions", sub, true, true)
	if r.Code != 200 {
		t.Fatal("selected destination could not be saved", r.Body.String())
	}
	if json.Unmarshal(r.Body.Bytes(), &sub) != nil {
		t.Fatal("subscription response")
	}
	stored, err := db.Subscription(context.Background(), sub.ID)
	if err != nil || stored.DestinationID != "two" || stored.Destination != "Anime/作品" || stored.DestinationAccountID != c.AccountID || stored.Rule().Renaming() {
		t.Fatal("folder identity or rename switch not saved", err)
	}
	if strings.Contains(r.Body.String(), c.AccountID) {
		t.Fatal("account ID exposed in subscription response")
	}
	c.AccountID = "another-private-account"
	if r := request("POST", "/api/pikpak/folders", create, true, true); r.Code != 400 || c.Calls["mkdir"] != 1 {
		t.Fatal("stale folder selection reused across accounts")
	}
	if r := request("PUT", "/api/subscriptions/"+strings.TrimSpace(jsonNumber(sub.ID)), sub, true, true); r.Code != 400 {
		t.Fatal("old account destination reused")
	}
	r = request("GET", "/api/pikpak/folders", nil, true, false)
	if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &listing) != nil {
		t.Fatal("new account listing")
	}
	c.Fail["list"] = errors.New("401 secret-response-sentinel")
	r = request("GET", "/api/pikpak/folders", nil, true, false)
	if r.Code != 400 || strings.Contains(r.Body.String(), "secret-response-sentinel") {
		t.Fatal("cloud error was not sanitized")
	}
}

func jsonNumber(value int64) string { b, _ := json.Marshal(value); return string(b) }
