package web

import (
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
)

func TestDeleteAndClearJobEndpoints(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	c := testutil.NewCloud()
	m := &folderManager{c}
	w := worker.New(db, m, feed.New(false))
	s, err := initializedServer(t, "x", store.AppSettings{}, db, m, w)
	if err != nil {
		t.Fatal(err)
	}
	h := s.Handler()
	s.sessions["fixture-session"] = session{until: time.Now().Add(time.Hour)}
	csrf := strings.Repeat("c", 43)
	request := func(path, body string, auth, token bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("DELETE", path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.AddCookie(&http.Cookie{Name: "pp_csrf", Value: csrf})
		if auth {
			r.AddCookie(&http.Cookie{Name: "pp_session", Value: "fixture-session"})
		}
		if token {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		return out
	}
	active := model.Job{ID: store.ID(), AccountID: c.AccountID, State: "downloading", TaskID: "remote"}
	complete := active
	complete.ID = store.ID()
	complete.State = "complete"
	for _, j := range []model.Job{active, complete} {
		if _, err := db.Enqueue(ctx, j, ""); err != nil {
			t.Fatal(err)
		}
	}
	c.Tasks["remote"] = pikpak.Task{ID: "remote", Status: "running"}
	c.Files["fixture"] = pikpak.File{ID: "fixture"}
	for _, path := range []string{"/api/jobs/" + active.ID, "/api/jobs/completed"} {
		if out := request(path, "", false, true); out.Code != 401 {
			t.Fatal("anonymous deletion accepted", out.Code)
		}
		if out := request(path, "", true, false); out.Code != 403 {
			t.Fatal("deletion without CSRF accepted", out.Code)
		}
	}
	if len(c.Calls) != 0 {
		t.Fatal("unauthorized requests reached cloud")
	}
	if out := request("/api/jobs/"+active.ID, `{"local_only":"invalid"}`, true, true); out.Code != 400 {
		t.Fatal("malformed deletion accepted")
	}
	c.Fail["cancel"] = &pikpak.Error{Kind: "auth", Message: "PikPak 授權失效，請更新 PAT"}
	if out := request("/api/jobs/"+active.ID, `{"local_only":true}`, true, true); out.Code != 400 {
		t.Fatal("cancellation failure ignored")
	}
	if _, err := db.Job(ctx, active.ID); err != nil {
		t.Fatal("failed cancellation lost local job")
	}
	if out := request("/api/jobs/"+active.ID, "{}", true, true); out.Code != 200 {
		t.Fatal(out.Code, out.Body.String())
	}
	if _, ok := c.Tasks["remote"]; ok {
		t.Fatal("remote task remains")
	}
	if _, ok := c.Files["fixture"]; !ok {
		t.Fatal("cloud file deleted")
	}
	if out := request("/api/jobs/"+active.ID, "", true, true); out.Code != 200 {
		t.Fatal("repeat delete failed")
	}
	out := request("/api/jobs/completed", "", true, true)
	var result struct {
		Deleted int `json:"deleted"`
	}
	if out.Code != 200 || json.Unmarshal(out.Body.Bytes(), &result) != nil || result.Deleted != 1 {
		t.Fatal("completed clear failed", out.Code, out.Body.String())
	}
	if c.Calls["cancel"] != 2 || c.Calls["submit"] != 0 || c.Calls["trash"] != 0 {
		t.Fatal("clear mutated cloud tasks/files")
	}
	page := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "pp_session", Value: "fixture-session"})
	h.ServeHTTP(page, r)
	if !strings.Contains(page.Body.String(), `id="clear-completed-jobs"`) {
		t.Fatal("clear-completed button missing")
	}
}
