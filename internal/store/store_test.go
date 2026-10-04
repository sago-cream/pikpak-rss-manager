package store

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/model"
)

func TestOptionalRenamingAndFolderIdentitySurviveRestart(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	off := false
	sub := model.Subscription{Name: "作品", RSSURL: "https://rss.test", RenameEnabled: off, DestinationID: "chosen-folder", DestinationAccountID: "private-owner", DestinationAccountRef: "ephemeral-ref"}
	if err := s.SaveSubscription(context.Background(), &sub); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Subscription(context.Background(), sub.ID)
	if err != nil || got.Rule().Renaming() || got.DestinationID != sub.DestinationID || got.DestinationAccountID != sub.DestinationAccountID || got.DestinationAccountRef != "" {
		t.Fatal("new subscription settings lost", err)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte(sub.DestinationAccountID)) {
		t.Fatal("private account ID escaped subscription API")
	}

}

func TestEncryptedPersistenceAndRepeatDownloads(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	token := "test-secret-token-123456"
	rss := "https://rss.test/feed?secret=test-private-rss"
	if err := s.SetSetting(ctx, "pikpak_token", token); err != nil {
		t.Fatal(err)
	}
	sub := model.Subscription{Name: "作品", RSSURL: rss}
	if err := s.SaveSubscription(ctx, &sub); err != nil {
		t.Fatal(err)
	}
	if err := s.Baseline(ctx, &sub, []string{"baseline"}); err != nil {
		t.Fatal(err)
	}
	j := model.Job{ID: ID(), SubscriptionID: sub.ID, AccountID: "account", ResourceKey: "btih:one", ResourceURL: "magnet:?secret=test-private-magnet", State: "queued"}
	added, err := s.Enqueue(ctx, j, "one")
	if err != nil || !added {
		t.Fatal(err)
	}
	j.ID = ID()
	added, err = s.Enqueue(ctx, j, "two")
	if err != nil || !added {
		t.Fatal("repeat resource was suppressed")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "manager.db"))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{token, rss, j.ResourceURL} {
		if bytes.Contains(b, []byte(v)) {
			t.Fatal("secret persisted in plaintext")
		}
	}
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Setting(ctx, "pikpak_token")
	if err != nil || got != token {
		t.Fatal("credential did not survive restart")
	}
	gotSub, err := s.Subscription(ctx, sub.ID)
	if err != nil || gotSub.ID != sub.ID || gotSub.RSSURL != rss || !gotSub.Initialized {
		t.Fatal("subscription did not survive restart", err)
	}
	seen, baseline, err := s.Seen(ctx, sub.ID, "baseline")
	if err != nil || !seen || !baseline {
		t.Fatal("initial baseline lost")
	}
	jobs, err := s.Jobs(ctx, 10)
	if err != nil || len(jobs) != 2 || jobs[0].AccountID != "account" || jobs[0].ResourceURL != j.ResourceURL {
		t.Fatal("private job state did not survive restart", err)
	}
}
