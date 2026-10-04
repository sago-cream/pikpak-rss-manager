// Package testutil contains a deterministic cloud double for failure-boundary
// tests. It deliberately does not perform network requests or read credentials.
package testutil

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

type Cloud struct {
	mu        sync.Mutex
	AccountID string
	Files     map[string]pikpak.File
	Tasks     map[string]pikpak.Task
	Fail      map[string]error // consumes one failure at the named boundary
	Calls     map[string]int
	OnSubmit  func(parent, source string) (pikpak.Task, error)
}

func NewCloud() *Cloud {
	return &Cloud{AccountID: "test-account", Files: map[string]pikpak.File{}, Tasks: map[string]pikpak.Task{}, Fail: map[string]error{}, Calls: map[string]int{}}
}
func (c *Cloud) boundary(name string) error {
	c.Calls[name]++
	err := c.Fail[name]
	delete(c.Fail, name)
	return err
}
func (c *Cloud) Account(context.Context) (pikpak.Account, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return pikpak.Account{ID: c.AccountID, Name: "測試帳號"}, c.boundary("account")
}
func (c *Cloud) List(_ context.Context, parent, token string) (pikpak.Page, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.boundary("list"); err != nil {
		return pikpak.Page{}, err
	}
	out := []pikpak.File{}
	for _, f := range c.Files {
		if f.ParentID == parent {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return pikpak.Page{Files: out}, nil
}
func (c *Cloud) Get(_ context.Context, id string) (pikpak.File, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.boundary("get"); err != nil {
		return pikpak.File{}, err
	}
	f, ok := c.Files[id]
	if !ok {
		return f, errors.New("test file missing")
	}
	return f, nil
}
func (c *Cloud) CreateFolder(_ context.Context, parent, name string) (pikpak.File, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.boundary("mkdir"); err != nil {
		return pikpak.File{}, err
	}
	f := pikpak.File{ID: store.ID(), ParentID: parent, Name: name, Kind: "drive#folder", Phase: "PHASE_TYPE_COMPLETE"}
	c.Files[f.ID] = f
	return f, nil
}
func (c *Cloud) Submit(_ context.Context, parent, source string) (pikpak.Task, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.boundary("submit"); err != nil {
		return pikpak.Task{}, err
	}
	if c.OnSubmit != nil {
		return c.OnSubmit(parent, source)
	}
	task := pikpak.Task{ID: store.ID(), Status: "running"}
	c.Tasks[task.ID] = task
	return task, nil
}
func (c *Cloud) Task(_ context.Context, id string) (pikpak.Task, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.boundary("task"); err != nil {
		return pikpak.Task{}, err
	}
	t, ok := c.Tasks[id]
	if !ok {
		return t, errors.New("test task missing")
	}
	return t, nil
}
func (c *Cloud) Rename(_ context.Context, id, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.boundary("rename"); err != nil {
		return err
	}
	f := c.Files[id]
	f.Name = name
	c.Files[id] = f
	return nil
}
func (c *Cloud) Move(_ context.Context, id, parent string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.boundary("move"); err != nil {
		return err
	}
	f := c.Files[id]
	f.ParentID = parent
	c.Files[id] = f
	return nil
}

func (c *Cloud) Trash(_ context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.boundary("trash"); err != nil {
		return err
	}
	for _, f := range c.Files {
		if f.ParentID == id {
			return errors.New("refusing to trash a nonempty folder")
		}
	}
	delete(c.Files, id)
	return nil
}
