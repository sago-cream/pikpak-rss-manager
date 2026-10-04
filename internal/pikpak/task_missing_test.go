package pikpak

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMissingTaskFromOfficialMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	var queries atomic.Int32
	for _, name := range []string{"account_info", "ls", "get", "mkdir", "add_link", "task_get", "rename"} {
		server.AddTool(&mcp.Tool{Name: name, InputSchema: map[string]any{"type": "object"}}, func(_ context.Context, r *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if r.Params.Name == "task_get" {
				queries.Add(1)
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "failed to get task: [REQUEST_FAILED] get task failed with status 404 private-url-secret"}}}, nil
			}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "{}"}}}, nil
		})
	}
	srv := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true}))
	defer srv.Close()
	c, err := newEndpoint(ctx, "fixture-token", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, err = c.Task(ctx, "missing-task")
	if err == nil || Classify(err).Kind != "not_found" || strings.Contains(err.Error(), "private-url-secret") || queries.Load() != 1 {
		t.Fatal("missing remote task must be classified without leaking its raw error or retrying", err)
	}
}

func TestNotFoundClassificationPreservesOtherFailures(t *testing.T) {
	for raw, want := range map[string]string{
		"get task failed with status 404": "not_found",
		"HTTP 404":                        "not_found",
		"404 Not Found":                   "not_found",
		"get task failed with status 500": "transient",
		"HTTP 401":                        "auth",
		"HTTP 429":                        "rate",
		"403 quota exceeded":              "quota",
	} {
		if got := Classify(errors.New(raw)); got.Kind != want || got.Message == raw {
			t.Fatalf("%q: got %s, want %s", raw, got.Kind, want)
		}
	}
}
