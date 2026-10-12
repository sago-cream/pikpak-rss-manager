package pikpak

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type storageReader struct {
	API
	read func(context.Context) (Account, error)
}

func (r *storageReader) Account(ctx context.Context) (Account, error) { return r.read(ctx) }

func storageAccount(id, used string) Account {
	a := Account{ID: id, Name: "Fixture"}
	a.Storage.Total = "1000"
	a.Storage.Used = used
	return a
}

func TestStorageRefreshCachesAndRetainsFailedReading(t *testing.T) {
	var calls int
	fail := false
	reader := &storageReader{read: func(ctx context.Context) (Account, error) {
		calls++
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 5*time.Second {
			t.Fatal("storage read not bounded")
		}
		if fail {
			return Account{}, errors.New("401 fixture credential must remain private")
		}
		return storageAccount("current", "250"), nil
	}}
	m := &Manager{api: reader, account: storageAccount("current", "100"), storageUpdatedAt: 1}
	got := m.RefreshStatus(context.Background())
	if got.StorageUsed != "250" || got.StorageUpdatedAt <= 1 {
		t.Fatal("reading not refreshed", got)
	}
	m.RefreshStatus(context.Background())
	if calls != 1 {
		t.Fatal("reading not cached")
	}
	fail = true
	m.storageAttempt = time.Now().Add(-2 * time.Minute)
	stale := m.RefreshStatus(context.Background())
	if stale.StorageUsed != got.StorageUsed || stale.StorageUpdatedAt != got.StorageUpdatedAt || stale.Paused != "" || stale.Message != "" {
		t.Fatal("failed read changed worker state or exposed provider error", stale)
	}
	m.RefreshStatus(context.Background())
	if calls != 2 {
		t.Fatal("failed reading retried too soon")
	}
	m.storageAttempt = time.Time{}
	m.paused = "auth"
	m.RefreshStatus(context.Background())
	if calls != 2 {
		t.Fatal("read while authorization paused")
	}
}

func TestStorageRefreshConcurrentReadersAndAccountSwitch(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	reader := &storageReader{read: func(context.Context) (Account, error) {
		calls.Add(1)
		close(started)
		<-release
		return storageAccount("old", "900"), nil
	}}
	m := &Manager{api: reader, account: storageAccount("old", "100"), storageUpdatedAt: 1}
	done := make(chan Status, 1)
	go func() { done <- m.RefreshStatus(context.Background()) }()
	<-started
	m.mu.Lock()
	m.storageAttempt = time.Now().Add(-2 * time.Minute)
	m.mu.Unlock()
	if got := m.RefreshStatus(context.Background()); got.StorageUsed != "100" || calls.Load() != 1 {
		t.Fatal("concurrent reader did not use cached snapshot", got)
	}
	m.mu.Lock()
	m.accountGeneration++
	m.account = storageAccount("new", "300")
	m.storageUpdatedAt = 2
	m.storageRefreshing = true
	m.mu.Unlock()
	close(release)
	got := <-done
	if got.StorageUsed != "300" || got.StorageUpdatedAt != 2 || !m.storageRefreshing {
		t.Fatal("old storage read overwrote switched account", got)
	}
}

func TestStorageRefreshRejectsMismatchedAccountAndKeepsQuotaPause(t *testing.T) {
	reader := &storageReader{read: func(context.Context) (Account, error) { return storageAccount("other", "999"), nil }}
	m := &Manager{api: reader, account: storageAccount("current", "100"), storageUpdatedAt: 1, paused: "quota"}
	got := m.RefreshStatus(context.Background())
	if got.StorageUsed != "100" || got.StorageUpdatedAt != 1 || got.Paused != "quota" {
		t.Fatal("mismatched account or quota state changed", got)
	}
}
