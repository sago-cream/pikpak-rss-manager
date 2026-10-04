package store

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/model"
)

func TestDeleteJobsPreservesBaselinesAndConfirmationIdempotency(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	sub := model.Subscription{Name: "Fixture", RSSURL: "https://fixture.test/rss"}
	if err := s.SaveSubscription(ctx, &sub); err != nil {
		t.Fatal(err)
	}
	j := model.Job{ID: ID(), SubscriptionID: sub.ID, AccountID: "fixture", ResourceKey: "btih:fixture", State: "downloading"}
	if _, err := s.Enqueue(ctx, j, "fingerprint"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveAction(ctx, model.FileAction{JobID: j.ID, FileID: "fixture", State: "pending"}); err != nil {
		t.Fatal(err)
	}
	if n, err := s.DeleteJob(ctx, j.ID); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if _, err := s.Job(ctx, j.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("deleted job remains", err)
	}
	if actions, err := s.Actions(ctx, j.ID); err != nil || len(actions) != 0 {
		t.Fatal("file actions remain", err)
	}
	if seen, _, err := s.Seen(ctx, sub.ID, "fingerprint"); err != nil || !seen {
		t.Fatal("deletion reset RSS baseline")
	}
	if n, err := s.DeleteJob(ctx, j.ID); err != nil || n != 0 {
		t.Fatal("repeat delete is not idempotent")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.EnqueueBatch(ctx, []model.Job{j}, []string{"fingerprint"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Job(ctx, j.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("confirmation retry resurrected deleted download")
	}
	j.ID = ID()
	if added, err := s.Enqueue(ctx, j, "explicit-new"); err != nil || !added {
		t.Fatal("explicit new download was suppressed")
	}
}

func TestClearCompletedJobsBeyondVisibleWindowAndRollback(t *testing.T) {
	ctx := context.Background()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	jobs := []model.Job{}
	fingerprints := []string{}
	for i := 0; i < 205; i++ {
		jobs = append(jobs, model.Job{ID: ID(), AccountID: "fixture", State: "complete"})
		fingerprints = append(fingerprints, "")
	}
	active := model.Job{ID: ID(), AccountID: "fixture", State: "downloading"}
	jobs = append(jobs, active)
	fingerprints = append(fingerprints, "")
	if err := s.EnqueueBatch(ctx, jobs, fingerprints); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveAction(ctx, model.FileAction{JobID: jobs[0].ID, FileID: "fixture", State: "done"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("CREATE TRIGGER block_delete BEFORE DELETE ON jobs BEGIN SELECT RAISE(ABORT,'fixture'); END"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClearCompletedJobs(ctx); err == nil {
		t.Fatal("injected delete failure was ignored")
	}
	if actions, err := s.Actions(ctx, jobs[0].ID); err != nil || len(actions) != 1 {
		t.Fatal("failed delete lost file actions")
	}
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM deleted_jobs").Scan(&n); err != nil || n != 0 {
		t.Fatal("failed delete retained tombstones")
	}
	if _, err := s.db.Exec("DROP TRIGGER block_delete"); err != nil {
		t.Fatal(err)
	}
	if n, err := s.ClearCompletedJobs(ctx); err != nil || n != 205 {
		t.Fatal("clear used the UI's 200-record window", n, err)
	}
	if _, err := s.Job(ctx, active.ID); err != nil {
		t.Fatal("clear removed an active download")
	}
	if n, err := s.ClearCompletedJobs(ctx); err != nil || n != 0 {
		t.Fatal("repeat clear is not idempotent")
	}
}
