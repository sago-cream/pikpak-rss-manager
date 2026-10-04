package pikpak

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

func TestOfficialSDKStreamableAdapter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	var mutations atomic.Int32
	for _, name := range []string{"account_info", "ls", "get", "mkdir", "add_link", "task_get", "rename"} {
		server.AddTool(&mcp.Tool{Name: name, InputSchema: map[string]any{"type": "object"}}, func(_ context.Context, r *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			text := "{}"
			switch r.Params.Name {
			case "account_info":
				text = `{"user_id":"test","name":"Test","storage":{"total":"100","used":"1"}}`
			case "ls":
				text = `{"files":[],"next_page_token":""}`
			case "add_link":
				mutations.Add(1)
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "429 too many requests; private-url-must-never-escape"}}}, nil
			case "task_get":
				text = `{"id":"task","status":"completed","progress":"100%","file_id":"file"}`
			}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil
		})
	}
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer unit-test-token" {
			w.WriteHeader(401)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	defer srv.Close()
	c, err := newEndpoint(ctx, "unit-test-token", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	account, err := c.Account(ctx)
	if err != nil || account.ID != "test" {
		t.Fatal("typed account response", err)
	}
	task, err := c.Task(ctx, "task")
	if err != nil || task.Percent() != 100 || task.State() != "completed" {
		t.Fatal("string percentage response", err)
	}
	_, err = c.Submit(ctx, "dest", "magnet:?test")
	if Classify(err).Kind != "rate" || mutations.Load() != 1 {
		t.Fatal("non-idempotent tool was repeated")
	}
	if err.Error() == "429 too many requests; private-url-must-never-escape" {
		t.Fatal("raw provider error escaped")
	}

}
func TestProgressAndErrorClassification(t *testing.T) {
	for _, status := range []string{"completed", "PHASE_TYPE_COMPLETE", "Complete (PHASE_TYPE_COMPLETE)"} {
		got := (Task{Status: status}).State()
		if got != "complete" && got != "completed" {
			t.Fatal("unrecognized complete status", got)
		}
	}
	for _, raw := range []string{`0`, `"0"`, `"50%"`, `100`, `""`, `null`} {
		var task Task
		if err := json.Unmarshal([]byte(`{"progress":`+raw+`}`), &task); err != nil {
			t.Fatal(raw, err)
		}
		if task.Percent() < 0 || task.Percent() > 100 {
			t.Fatal("invalid percentage")
		}
	}
	for text, want := range map[string]string{"HTTP 401 private token": "auth", "quota exceeded": "quota", "403 cloud download quota exceeded": "quota", "403 storage_space_limit": "quota", "not enough space": "quota", "空間不足": "quota", "429 throttled": "rate", "connection lost private url": "transient"} {
		if got := Classify(errors.New(text)); got.Kind != want || got.Message == text {
			t.Fatal("classification leaked raw response", got)
		}
	}
}
func TestAccountSwitchPausesDurableIntent(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sub := model.Subscription{Name: "Test", RSSURL: "https://rss.test"}
	_ = db.SaveSubscription(ctx, &sub)
	j := model.Job{ID: store.ID(), SubscriptionID: sub.ID, AccountID: "old", ResourceKey: "one", ResourceURL: "magnet:?one", State: "submitting"}
	if _, err := db.Enqueue(ctx, j, "one"); err != nil {
		t.Fatal(err)
	}
	m := NewManager(db)
	m.factory = func(context.Context, string) (API, error) {
		return &accountOnly{value: Account{ID: "new", Name: "New"}}, nil
	}
	if err := m.Bind(ctx, "unit-token"); err != nil {
		t.Fatal(err)
	}
	got, _ := db.Job(ctx, j.ID)
	if got.State != "paused_account" {
		t.Fatal("old in-flight intent remained runnable")
	}
	m.factory = func(context.Context, string) (API, error) { return nil, &Error{Kind: "auth", Message: "invalid token"} }
	if err := m.Bind(ctx, "invalid-candidate"); err == nil {
		t.Fatal("invalid candidate accepted")
	}
	api, account := m.Snapshot()
	if api == nil || account != "new" {
		t.Fatal("failed replacement disrupted valid account")
	}
}

type accountOnly struct {
	API
	value Account
}

func (a *accountOnly) Account(context.Context) (Account, error) { return a.value, nil }
