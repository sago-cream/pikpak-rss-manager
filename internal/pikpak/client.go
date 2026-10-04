package pikpak

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const Endpoint = "https://pikpak.ai/mcp"

type Account struct {
	ID      string `json:"user_id"`
	Name    string `json:"name"`
	Storage struct {
		Total string `json:"total"`
		Used  string `json:"used"`
	} `json:"storage"`
}
type File struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	ParentID string `json:"parent_id"`
	Phase    string `json:"phase"`
	Size     string `json:"size"`
	MIME     string `json:"mime_type"`
	Trashed  bool   `json:"trashed"`
}

func (f File) Folder() bool { return f.Kind == "drive#folder" || f.Kind == "folder" }

type Page struct {
	Files []File `json:"files"`
	Next  string `json:"next_page_token"`
}
type Progress string

func (p *Progress) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*p = ""
		return nil
	}
	var text string
	if len(b) > 0 && b[0] == '"' {
		if err := json.Unmarshal(b, &text); err != nil {
			return err
		}
	} else {
		var n json.Number
		if err := json.Unmarshal(b, &n); err != nil {
			return err
		}
		text = n.String()
	}
	*p = Progress(strings.TrimSuffix(strings.TrimSpace(text), "%"))
	return nil
}

type Task struct {
	ID       string   `json:"id"`
	TaskID   string   `json:"task_id"`
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Phase    string   `json:"phase"`
	FileID   string   `json:"file_id"`
	Progress Progress `json:"progress"`
	Message  string   `json:"message"`
}

func (t Task) State() string {
	state := t.Status
	if state == "" {
		state = t.Phase
	}
	state = strings.ToLower(strings.TrimSpace(state))
	// The hosted MCP returns labels such as "Complete (PHASE_TYPE_COMPLETE)".
	if i := strings.Index(state, "phase_type_"); i >= 0 {
		state = state[i+len("phase_type_"):]
		if end := strings.IndexAny(state, ") \t\r\n"); end >= 0 {
			state = state[:end]
		}
		return state
	}
	return state
}
func (t Task) Percent() float64 {
	f, _ := strconv.ParseFloat(string(t.Progress), 64)
	if f < 0 {
		return 0
	}
	if f > 100 {
		return 100
	}
	return f
}

type API interface {
	Account(context.Context) (Account, error)
	List(context.Context, string, string) (Page, error)
	Get(context.Context, string) (File, error)
	CreateFolder(context.Context, string, string) (File, error)
	Submit(context.Context, string, string) (Task, error)
	Task(context.Context, string) (Task, error)
	Rename(context.Context, string, string) error
}
type Error struct {
	Kind    string
	Message string
}

func (e *Error) Error() string { return e.Message }
func Classify(err error) *Error {
	var known *Error
	if errors.As(err, &known) {
		return known
	}
	if err == nil {
		return nil
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "401") || strings.Contains(text, "unauthorized") || strings.Contains(text, "token expired") || strings.Contains(text, "invalid_token"):
		return &Error{"auth", "PikPak 授權失效，請更新 PAT"}
	case strings.Contains(text, "quota") || strings.Contains(text, "space not enough") || strings.Contains(text, "not enough space") || strings.Contains(text, "storage_limit") || strings.Contains(text, "storage_space_limit") || strings.Contains(text, "cloud_download_limit") || strings.Contains(text, "空间不足") || strings.Contains(text, "空間不足"):
		return &Error{"quota", "PikPak 配額不足；任務已暫停"}
	case strings.Contains(text, "403") || strings.Contains(text, "permission") || strings.Contains(text, "scope"):
		return &Error{"auth", "PikPak 權限不足，請確認檔案讀寫及雲端下載權限"}
	case strings.Contains(text, "429") || strings.Contains(text, "rate limit") || strings.Contains(text, "too many"):
		return &Error{"rate", "PikPak 請求受到限流，稍後重試"}
	case strings.Contains(text, "status 404") || strings.Contains(text, "http 404") || strings.Contains(text, "404 not found"):
		return &Error{"not_found", "PikPak 項目不存在，請至 PikPak 檢查"}
	default:
		return &Error{"transient", "PikPak 請求失敗或逾時"}
	}
}

type bearerTransport struct {
	token    string
	base     http.RoundTripper
	endpoint string
}

func (t bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.String() != t.endpoint {
		return nil, &Error{"permanent", "拒絕將權杖送到其他端點"}
	}
	copy := r.Clone(r.Context())
	copy.Header = r.Header.Clone()
	copy.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(copy)
}

type Client struct {
	mu              sync.Mutex
	session         *mcp.ClientSession
	token, endpoint string
	lastCall        time.Time
	canCancel       bool
}

func New(ctx context.Context, token string) (*Client, error) {
	return newEndpoint(ctx, token, Endpoint)
}
func newEndpoint(ctx context.Context, token, endpoint string) (*Client, error) {
	if strings.TrimSpace(token) == "" || strings.ContainsAny(token, "\r\n") {
		return nil, &Error{"auth", "PAT 為空或格式不正確"}
	}
	c := &Client{token: token, endpoint: endpoint}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connect(ctx); err != nil {
		return nil, err
	}
	required := map[string]bool{}
	for _, tool := range []string{"account_info", "ls", "get", "mkdir", "add_link", "task_get", "rename"} {
		required[tool] = false
	}
	for tool, err := range c.session.Tools(ctx, nil) {
		if err != nil {
			c.session.Close()
			return nil, Classify(err)
		}
		if _, ok := required[tool.Name]; ok {
			required[tool.Name] = true
		}
		if tool.Name == "task_rm" {
			c.canCancel = true
		}
	}
	for name, found := range required {
		if !found {
			c.session.Close()
			return nil, &Error{"permanent", fmt.Sprintf("官方 MCP 缺少必要工具 %s", name)}
		}
	}
	return c, nil
}
func (c *Client) connect(ctx context.Context) error {
	hc := &http.Client{Timeout: 25 * time.Second, Transport: bearerTransport{c.token, http.DefaultTransport, c.endpoint}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "pikpak-rss-manager", Version: "0.1.0"}, nil).Connect(ctx, &mcp.StreamableClientTransport{Endpoint: c.endpoint, HTTPClient: hc, MaxRetries: -1, DisableStandaloneSSE: true, MaxEventSize: 4 << 20}, nil)
	if err != nil {
		return Classify(err)
	}
	c.session = session
	return nil
}
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}
func (c *Client) call(ctx context.Context, name string, args map[string]any, out any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if wait := time.Until(c.lastCall.Add(300 * time.Millisecond)); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return Classify(ctx.Err())
		case <-timer.C:
		}
	}
	if c.session == nil {
		if err := c.connect(ctx); err != nil {
			return err
		}
	}
	c.lastCall = time.Now()
	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		_ = c.session.Close()
		c.session = nil
		return Classify(err)
	}
	if result.IsError {
		var text string
		for _, content := range result.Content {
			if t, ok := content.(*mcp.TextContent); ok {
				text += t.Text
			}
		}
		return Classify(errors.New(text))
	}
	if out == nil {
		return nil
	}
	var payload []byte
	if result.StructuredContent != nil {
		payload, err = json.Marshal(result.StructuredContent)
	} else {
		for _, content := range result.Content {
			if t, ok := content.(*mcp.TextContent); ok {
				payload = []byte(t.Text)
				break
			}
		}
	}
	if err != nil || len(payload) == 0 || len(payload) > 4<<20 {
		return &Error{"permanent", "官方 MCP 回傳資料格式無法解析"}
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	if err := decoder.Decode(out); err != nil {
		return &Error{"permanent", "官方 MCP 回傳資料格式與預期不同"}
	}
	return nil
}
func (c *Client) Account(ctx context.Context) (Account, error) {
	var a Account
	err := c.call(ctx, "account_info", map[string]any{}, &a)
	if err == nil && a.ID == "" {
		err = &Error{"permanent", "官方 MCP 沒有回傳帳號識別"}
	}
	return a, err
}
func (c *Client) List(ctx context.Context, parent, token string) (Page, error) {
	var p Page
	err := c.call(ctx, "ls", map[string]any{"parent_id": parent, "token": token, "limit": 100, "raw": true}, &p)
	return p, err
}
func (c *Client) Get(ctx context.Context, id string) (File, error) {
	var f File
	err := c.call(ctx, "get", map[string]any{"id": id, "raw": true}, &f)
	return f, err
}
func (c *Client) CreateFolder(ctx context.Context, parent, name string) (File, error) {
	var f File
	err := c.call(ctx, "mkdir", map[string]any{"parent": parent, "name": name}, &f)
	if err == nil && f.ID == "" {
		err = &Error{"transient", "建立目錄結果不明，請重新核對"}
	}
	return f, err
}
func (c *Client) Submit(ctx context.Context, parent, source string) (Task, error) {
	var t Task
	err := c.call(ctx, "add_link", map[string]any{"parent": parent, "url": source}, &t)
	if t.ID == "" {
		t.ID = t.TaskID
	}
	if err == nil && t.ID == "" {
		err = &Error{"transient", "離線提交結果不明，請核對 PikPak 任務"}
	}
	return t, err
}
func (c *Client) Task(ctx context.Context, id string) (Task, error) {
	var t Task
	err := c.call(ctx, "task_get", map[string]any{"id": id}, &t)
	return t, err
}
func (c *Client) Rename(ctx context.Context, id, name string) error {
	return c.call(ctx, "rename", map[string]any{"id": id, "name": name}, nil)
}

func ListAll(ctx context.Context, api API, parent string) ([]File, error) {
	out := []File{}
	token := ""
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		p, err := api.List(ctx, parent, token)
		if err != nil {
			return nil, err
		}
		out = append(out, p.Files...)
		if p.Next == "" {
			return out, nil
		}
		if seen[p.Next] {
			return nil, &Error{"permanent", "目錄分頁重複，停止處理"}
		}
		seen[p.Next] = true
		token = p.Next
	}
	return nil, &Error{"permanent", "目錄超過處理上限"}
}
func EnsureFolder(ctx context.Context, api API, parent, name string) (string, error) {
	files, err := ListAll(ctx, api, parent)
	if err != nil {
		return "", err
	}
	found := ""
	for _, f := range files {
		if f.Name == name {
			if !f.Folder() || found != "" {
				return "", &Error{"permanent", "目標目錄有同名檔案或多個同名目錄，請先整理"}
			}
			found = f.ID
		}
	}
	if found != "" {
		return found, nil
	}
	f, err := api.CreateFolder(ctx, parent, name)
	return f.ID, err
}
