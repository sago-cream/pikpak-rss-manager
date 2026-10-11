package store

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRecoveryExpiryReplacementBindingAndConsumption(t *testing.T) {
	ctx, now := context.Background(), time.Now()
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.IssuePasswordRecovery(ctx, now); err == nil {
		t.Fatal("issued recovery before initialization")
	}
	settings := AppSettings{PublicURL: "https://fixture.test", AllowPrivateFeeds: true}
	if _, err := db.InitializeAdministrator(ctx, "initial-verifier", settings); err != nil {
		t.Fatal(err)
	}
	if err := db.SetSetting(ctx, "pikpak_token", "fixture-pat"); err != nil {
		t.Fatal(err)
	}
	first, err := db.IssuePasswordRecovery(ctx, now)
	if err != nil || len(first) != 43 {
		t.Fatal("recovery code issuance failed", err)
	}
	var raw string
	if err := db.db.QueryRow("SELECT value FROM settings WHERE key=?", recoverySetting).Scan(&raw); err != nil || !strings.HasPrefix(raw, "enc:") || strings.Contains(raw, first) {
		t.Fatal("recovery record is not encrypted or contains plaintext code")
	}
	decoded, _ := db.Setting(ctx, recoverySetting)
	if strings.Contains(decoded, first) {
		t.Fatal("persisted plaintext recovery code")
	}
	for _, tc := range []struct {
		code string
		when time.Time
	}{
		{"incorrect-code", now},
		{first, now.Add(PasswordRecoveryTTL)},
	} {
		if err := db.RecoverAdministratorPassword(ctx, tc.code, "replacement", tc.when); !errors.Is(err, ErrRecoveryCode) {
			t.Fatal("invalid or expired code accepted", err)
		}
	}
	second, err := db.IssuePasswordRecovery(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RecoverAdministratorPassword(ctx, first, "replacement", now); !errors.Is(err, ErrRecoveryCode) {
		t.Fatal("reissued code did not revoke old code")
	}
	if err := db.ChangeAdministratorPassword(ctx, "initial-verifier", "changed-verifier"); err != nil {
		t.Fatal(err)
	}
	if err := db.RecoverAdministratorPassword(ctx, second, "replacement", now); !errors.Is(err, ErrRecoveryCode) {
		t.Fatal("password change did not revoke recovery code")
	}
	code, err := db.IssuePasswordRecovery(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	winners := 0
	for range 4 {
		wg.Go(func() {
			err := db.RecoverAdministratorPassword(ctx, code, "recovered-verifier", now)
			if err == nil {
				mu.Lock()
				winners++
				mu.Unlock()
			} else if !errors.Is(err, ErrRecoveryCode) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if winners != 1 {
		t.Fatal("code was not consumed exactly once", winners)
	}
	hash, got, err := db.Administrator(ctx)
	if err != nil || hash != "recovered-verifier" || got != settings {
		t.Fatal("recovery changed site settings or lost verifier", err)
	}
	token, err := db.Setting(ctx, "pikpak_token")
	if err != nil || token != "fixture-pat" {
		t.Fatal("recovery changed cloud credentials", err)
	}
	if err := db.RecoverAdministratorPassword(ctx, code, "replay", now); !errors.Is(err, ErrRecoveryCode) {
		t.Fatal("consumed recovery code was replayed")
	}
}
