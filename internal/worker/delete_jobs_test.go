package worker

import (
	"context"
	"testing"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
)

func TestDeleteJobCancelsTaskAndPreservesCloudFiles(t *testing.T) {
	ctx := context.Background()
	w, c, _, _, sub, _ := setup(t)
	j := enqueue(t, w, sub, "downloading")
	j.TaskID = "remote"
	if err := w.DB.SaveJob(ctx, &j); err != nil {
		t.Fatal(err)
	}
	c.Tasks["remote"] = pikpak.Task{ID: "remote", Status: "running"}
	c.Files["downloaded"] = pikpak.File{ID: "downloaded", Name: "fixture.txt"}
	c.Fail["cancel"] = &pikpak.Error{Kind: "transient", Message: "cancel failed"}
	if _, err := w.DeleteJob(ctx, j.ID, false); err == nil {
		t.Fatal("failed cancellation removed job")
	}
	if _, err := w.DB.Job(ctx, j.ID); err != nil {
		t.Fatal("failed cancellation lost recovery state")
	}
	if n, err := w.DeleteJob(ctx, j.ID, false); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if _, ok := c.Tasks["remote"]; ok {
		t.Fatal("remote task was not cancelled")
	}
	if _, ok := c.Files["downloaded"]; !ok {
		t.Fatal("downloaded cloud file was removed")
	}
	if c.Calls["submit"] != 0 || c.Calls["trash"] != 0 || c.Calls["move"] != 0 {
		t.Fatal("deletion changed cloud content")
	}
	if n, err := w.DeleteJob(ctx, j.ID, false); err != nil || n != 0 {
		t.Fatal("repeat deletion failed")
	}
}

func TestDeleteJobMissingTaskUnknownSubmissionAndAccountSwitch(t *testing.T) {
	ctx := context.Background()
	w, c, _, _, sub, _ := setup(t)
	j := enqueue(t, w, sub, "submission_unknown")
	if _, err := w.DeleteJob(ctx, j.ID, false); err == nil {
		t.Fatal("unknown submission was silently claimed cancelled")
	}
	if n, err := w.DeleteJob(ctx, j.ID, true); err != nil || n != 1 {
		t.Fatal("confirmed local-only removal failed")
	}
	j = enqueue(t, w, sub, "downloading")
	j.TaskID = "gone"
	if err := w.DB.SaveJob(ctx, &j); err != nil {
		t.Fatal(err)
	}
	c.Fail["task"] = &pikpak.Error{Kind: "not_found", Message: "gone"}
	if n, err := w.DeleteJob(ctx, j.ID, false); err != nil || n != 1 || c.Calls["cancel"] != 0 {
		t.Fatal("already removed task could not be deleted")
	}
	j = enqueue(t, w, sub, "downloading")
	j.TaskID = "old-task"
	if err := w.DB.SaveJob(ctx, &j); err != nil {
		t.Fatal(err)
	}
	c.AccountID = "another-account"
	if _, err := w.DeleteJob(ctx, j.ID, false); err == nil {
		t.Fatal("cancelled task on another account")
	}
	if _, err := w.DB.Job(ctx, j.ID); err != nil {
		t.Fatal("account switch lost job")
	}
}

func TestDeleteSerializesWithCloudSubmission(t *testing.T) {
	ctx := context.Background()
	w, c, _, _, sub, _ := setup(t)
	j := enqueue(t, w, sub, "queued")
	entered := make(chan struct{})
	release := make(chan struct{})
	processed := make(chan error, 1)
	deleted := make(chan error, 1)
	c.OnSubmit = func(_, _ string) (pikpak.Task, error) {
		close(entered)
		<-release
		c.Tasks["remote"] = pikpak.Task{ID: "remote", Status: "running"}
		return c.Tasks["remote"], nil
	}
	go func() { processed <- w.Process(ctx, j.ID) }()
	<-entered
	go func() { _, err := w.DeleteJob(ctx, j.ID, false); deleted <- err }()
	select {
	case err := <-deleted:
		close(release)
		t.Fatal("deletion raced with submission", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	if err := <-processed; err != nil {
		t.Fatal(err)
	}
	if err := <-deleted; err != nil {
		t.Fatal(err)
	}
	if c.Calls["submit"] != 1 || c.Calls["cancel"] != 1 {
		t.Fatal("in-flight submission was orphaned")
	}
	if err := w.Process(ctx, j.ID); err == nil {
		t.Fatal("deleted job was still processable")
	}
	if c.Calls["submit"] != 1 {
		t.Fatal("deleted job resubmitted")
	}
}

func TestClearCompletedPreservesActiveAndDoesNotUseCloud(t *testing.T) {
	ctx := context.Background()
	w, c, p, _, sub, _ := setup(t)
	enqueue(t, w, sub, "complete")
	active := enqueue(t, w, sub, "downloading")
	p.paused = &pikpak.Error{Kind: "auth", Message: "fixture"}
	if n, err := w.ClearCompletedJobs(ctx); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if _, err := w.DB.Job(ctx, active.ID); err != nil {
		t.Fatal(err)
	}
	if len(c.Calls) != 0 {
		t.Fatal("clear contacted cloud")
	}
}
