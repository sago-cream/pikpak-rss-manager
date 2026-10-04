package pikpak

import (
	"context"
	"errors"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"sync"
)

type Status struct {
	Connected    bool   `json:"connected"`
	Name         string `json:"name"`
	Source       string `json:"source"`
	External     bool   `json:"external"`
	Paused       string `json:"paused"`
	Message      string `json:"message"`
	StorageTotal string `json:"storage_total"`
	StorageUsed  string `json:"storage_used"`
}
type Manager struct {
	mu                             sync.RWMutex
	db                             *store.Store
	api                            API
	account                        Account
	token, source, paused, message string
	factory                        func(context.Context, string) (API, error)
}

func NewManager(db *store.Store) *Manager {
	return &Manager{db: db, factory: func(ctx context.Context, t string) (API, error) { return New(ctx, t) }}
}
func (m *Manager) Initialize(ctx context.Context) error {
	m.mu.RLock()
	token := m.token
	m.mu.RUnlock()
	if token == "" {
		t, err := m.db.Setting(ctx, "pikpak_token")
		if err != nil {
			return err
		}
		if t == "" {
			return nil
		}
		m.mu.Lock()
		m.token = t
		m.source = "database"
		m.mu.Unlock()
	}
	return m.Reconnect(ctx)
}
func (m *Manager) Reconnect(ctx context.Context) error {
	m.mu.RLock()
	token := m.token
	m.mu.RUnlock()
	if token == "" {
		return errors.New("尚未綁定 PikPak PAT")
	}
	return m.bind(ctx, token, false)
}
func (m *Manager) Bind(ctx context.Context, token string) error {
	if token == "" {
		return errors.New("請輸入 PikPak PAT")
	}
	return m.bind(ctx, token, true)
}
func (m *Manager) bind(ctx context.Context, token string, save bool) error {
	api, err := m.factory(ctx, token)
	if err != nil {
		if !save {
			m.Pause(Classify(err))
		}
		return Classify(err)
	}
	account, err := api.Account(ctx)
	if err != nil {
		if c, ok := api.(interface{ Close() error }); ok {
			_ = c.Close()
		}
		if !save {
			m.Pause(Classify(err))
		}
		return Classify(err)
	}
	if save {
		if err := m.db.SetSetting(ctx, "pikpak_token", token); err != nil {
			if c, ok := api.(interface{ Close() error }); ok {
				_ = c.Close()
			}
			return errors.New("無法儲存 PAT")
		}
	}
	m.mu.Lock()
	old := m.api
	m.api = api
	m.account = account
	m.token = token
	m.paused = ""
	m.message = ""
	if save {
		m.source = "database"
	}
	m.mu.Unlock()
	if old != nil {
		if c, ok := old.(interface{ Close() error }); ok {
			_ = c.Close()
		}
	}
	jobs, err := m.db.Jobs(ctx, 10000)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if j.AccountID != account.ID && (j.State == "queued" || j.State == "submitting" || j.State == "downloading" || j.State == "organizing") {
			j.State = "paused_account"
			j.Error = "目前綁定不同帳號；舊帳號任務已暫停"
			if err := m.db.SaveJob(ctx, &j); err != nil {
				return err
			}
		}
	}
	return nil
}
func (m *Manager) Snapshot() (API, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.paused != "" {
		return nil, m.account.ID
	}
	return m.api, m.account.ID
}
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Status{m.api != nil, m.account.Name, m.source, false, m.paused, m.message, m.account.Storage.Total, m.account.Storage.Used}
}
func (m *Manager) Pause(e *Error) {
	if e == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.message = e.Message
	if e.Kind == "auth" || e.Kind == "quota" {
		m.paused = e.Kind
	}
}
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.api.(interface{ Close() error }); ok {
		_ = c.Close()
	}
}
