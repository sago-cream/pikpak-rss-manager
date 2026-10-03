package store

import (
	"context"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/model"
)

func TestRepeatDownloadMigrationPreservesJobActionsAndEncryption(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	sub := model.Subscription{Name: "Fixture", RSSURL: "https://fixture.test/feed?private=fixture"}
	if err := s.SaveSubscription(ctx, &sub); err != nil {
		t.Fatal(err)
	}
	j := model.Job{ID: ID(), SubscriptionID: sub.ID, AccountID: "fixture", ResourceKey: "btih:fixture", ResourceURL: "magnet:?xt=private-fixture", State: "organizing", StagingID: "persisted-stage", TaskID: "persisted-task"}
	if _, err := s.Enqueue(ctx, j, "seen"); err != nil {
		t.Fatal(err)
	}
	a := model.FileAction{JobID: j.ID, FileID: "file", State: "renamed", TargetName: "Fixture.mkv"}
	if err := s.SaveAction(ctx, a); err != nil {
		t.Fatal(err)
	}
	// Model the version-2 uniqueness index before reopening/migrating.
	if _, err := s.db.Exec("CREATE UNIQUE INDEX old_resource_unique ON jobs(account_id,resource_key); PRAGMA user_version=2"); err != nil {
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
	got, err := s.Job(ctx, j.ID)
	if err != nil || got.ResourceURL != j.ResourceURL || got.StagingID != j.StagingID || got.TaskID != j.TaskID {
		t.Fatal("migration lost private job or recovery IDs", err)
	}
	actions, err := s.Actions(ctx, j.ID)
	if err != nil || len(actions) != 1 || actions[0].State != "renamed" {
		t.Fatal("migration lost partial action", err)
	}
	rows, err := s.db.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	if rows.Next() {
		t.Fatal("migration broke foreign keys")
	}
	rows.Close()
	j.ID = ID()
	if added, err := s.Enqueue(ctx, j, "second"); err != nil || !added {
		t.Fatal("repeat resource suppressed after migration", err)
	}
	if added, err := s.Enqueue(ctx, j, "second"); err != nil || added {
		t.Fatal("same request ID duplicated", err)
	}
}

func TestSelectionBatchIsAtomicAndIdempotent(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	sub := model.Subscription{Name: "Fixture", RSSURL: "https://fixture.test"}
	if err := s.SaveSubscription(ctx, &sub); err != nil {
		t.Fatal(err)
	}
	jobs := []model.Job{{ID: ID(), AccountID: "fixture", ResourceKey: "same", State: "queued", SubscriptionID: sub.ID}, {ID: ID(), AccountID: "fixture", ResourceKey: "same", State: "queued", SubscriptionID: 999}}
	if err := s.EnqueueBatch(ctx, jobs, []string{"one", "two"}); err == nil {
		t.Fatal("invalid batch committed")
	}
	all, _ := s.Jobs(ctx, 10)
	if len(all) != 0 {
		t.Fatal("partial batch committed")
	}
	jobs[1].SubscriptionID = sub.ID
	for range 2 {
		if err := s.EnqueueBatch(ctx, jobs, []string{"one", "two"}); err != nil {
			t.Fatal(err)
		}
	}
	all, _ = s.Jobs(ctx, 10)
	if len(all) != 2 {
		t.Fatal("batch replay repeated or suppressed selected jobs")
	}
}
