package pikpak

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestOfficialTaskCancellationPreservesFilesAndDoesNotRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server := mcp.NewServer(&mcp.Implementation{Name: "fixture", Version: "1"}, nil)
	var calls atomic.Int32
	for _, name := range []string{"account_info", "ls", "get", "mkdir", "add_link", "task_get", "rename", "mv", "task_rm"} {
		server.AddTool(&mcp.Tool{Name: name, InputSchema: map[string]any{"type": "object"}}, func(_ context.Context, r *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if r.Params.Name == "task_rm" {
				var args struct {
					IDs         []string `json:"ids"`
					DeleteFiles *bool    `json:"delete-files"`
				}
				if err := json.Unmarshal(r.Params.Arguments, &args); err != nil || len(args.IDs) != 1 || args.IDs[0] != "fixture-task" || args.DeleteFiles == nil || *args.DeleteFiles {
					t.Error("task cancellation must explicitly preserve files")
				}
				if calls.Add(1) == 2 {
					return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "503 private-provider-url"}}}, nil
				}
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
	if err := c.CancelTask(ctx, "fixture-task"); err != nil {
		t.Fatal(err)
	}
	if err := c.CancelTask(ctx, "fixture-task"); err == nil || strings.Contains(err.Error(), "private-provider-url") || calls.Load() != 2 {
		t.Fatal("uncertain cancellation was retried or leaked raw response", err)
	}
}
