package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
)

func TestAdministratorAtomicSetupAndMigration(t *testing.T) {
	dir := t.TempDir()
	// Build a v1 database with a private setting before opening the current
	// version, proving the migration preserves existing durable state.
	legacy, err := sql.Open("sqlite", filepath.Join(dir, "manager.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec(migration); err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec("INSERT INTO events(level,message,created_at) VALUES('info','legacy-event',1)"); err != nil {
		t.Fatal(err)
	}
	legacy.Close()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	second, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	ctx := context.Background()
	hash, _, err := s.Administrator(ctx)
	if err != nil || hash != "" {
		t.Fatal("migration auto-created credentials")
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	var winners int
	var mu sync.Mutex
	for _, db := range []*Store{s, second} {
		wg.Add(1)
		go func(db *Store) {
			defer wg.Done()
			<-start
			created, err := db.InitializeAdministrator(ctx, "fixture-verifier", AppSettings{PublicURL: "https://panel.test", AllowPrivateFeeds: true})
			if err != nil {
				t.Error(err)
			}
			if created {
				mu.Lock()
				winners++
				mu.Unlock()
			}
		}(db)
	}
	close(start)
	wg.Wait()
	if winners != 1 {
		t.Fatal("concurrent setup did not have exactly one winner", winners)
	}
	if created, err := s.InitializeAdministrator(ctx, "replacement-verifier", AppSettings{}); err != nil || created {
		t.Fatal("setup overwrote administrator")
	}
	hash, settings, err := second.Administrator(ctx)
	if err != nil || hash != "fixture-verifier" || !settings.AllowPrivateFeeds || settings.PublicURL != "https://panel.test" {
		t.Fatal("setup values lost", err)
	}
	var count, version int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM events WHERE message='legacy-event'").Scan(&count); err != nil || count != 1 {
		t.Fatal("migration lost existing state")
	}
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 3 {
		t.Fatal("migration version incorrect")
	}
}

func TestAdministratorPasswordRejectsStaleVerifier(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	other, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	ctx := context.Background()
	if err := s.ChangeAdministratorPassword(ctx, "original-verifier", "new-verifier"); err == nil {
		t.Fatal("password change created an administrator before setup")
	}
	settings := AppSettings{PublicURL: "https://panel.test", AllowPrivateFeeds: true}
	if _, err := s.InitializeAdministrator(ctx, "original-verifier", settings); err != nil {
		t.Fatal(err)
	}
	if err := s.ChangeAdministratorPassword(ctx, "original-verifier", ""); err == nil {
		t.Fatal("empty verifier accepted")
	}
	if err := s.ChangeAdministratorPassword(ctx, "original-verifier", "new-verifier"); err != nil {
		t.Fatal(err)
	}
	if err := other.ChangeAdministratorPassword(ctx, "original-verifier", "stale-verifier"); err == nil {
		t.Fatal("stale change overwrote password")
	}
	hash, got, err := other.Administrator(ctx)
	if err != nil || hash != "new-verifier" || got != settings {
		t.Fatal("password change lost credentials or settings", err)
	}
}
