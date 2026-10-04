package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/testutil"
)

func directFixture(t *testing.T) (*Worker, *testutil.Cloud, model.Job, string) {
	t.Helper()
	w, c, _, _, sub, dir := setup(t)
	on := true
	sub.RenameEnabled, sub.RenameMode = &on, "replace"
	sub.Regex, sub.Replacement = `^old`, "new"
	if err := w.SaveSubscription(context.Background(), &sub); err != nil {
		t.Fatal(err)
	}
	j := enqueue(t, w, sub, "organizing")
	j.DownloadMode, j.DestinationID, j.FileID = "direct", "dest", "media"
	c.Files["dest"] = pikpak.File{ID: "dest", Name: "Downloads", Kind: "drive#folder"}
	c.Files["media"] = pikpak.File{ID: "media", ParentID: "dest", Name: "old.mkv", Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
	if err := w.DB.SaveJob(context.Background(), &j); err != nil {
		t.Fatal(err)
	}
	return w, c, j, dir
}

func assertDirectOnly(t *testing.T, c *testutil.Cloud) {
	t.Helper()
	if c.Calls["move"] != 0 || c.Calls["trash"] != 0 || c.Calls["mkdir"] != 0 {
		t.Fatal("direct task mutated cloud structure", c.Calls)
	}
}

func TestDirectSubmissionAndNoRename(t *testing.T) {
	for _, parent := range []string{"", "dest"} {
		t.Run("parent="+parent, func(t *testing.T) {
			w, c, j, _ := directFixture(t)
			off := false
			j.Rule.RenameEnabled, j.DestinationID, j.Destination, j.FileID, j.State = &off, parent, "", "", "queued"
			c.OnSubmit = func(got, source string) (pikpak.Task, error) {
				if got != parent {
					t.Fatal("wrong submission parent", got)
				}
				c.Tasks["task"] = pikpak.Task{ID: "task", Status: "complete"}
				return pikpak.Task{ID: "task", FileID: "media"}, nil
			}
			ctx := context.Background()
			if err := w.DB.SaveJob(ctx, &j); err != nil {
				t.Fatal(err)
			}
			if err := w.Process(ctx, j.ID); err != nil {
				t.Fatal(err)
			}
			if got := saved(t, w, j); got.FileID != "media" || got.StagingID != "" || got.StagingCleanup != nil {
				t.Fatal("submission IDs not saved", got)
			}
			if err := w.Process(ctx, j.ID); err != nil {
				t.Fatal(err)
			}
			if got := saved(t, w, j); got.State != "complete" || c.Calls["rename"] != 0 || c.Calls["submit"] != 1 {
				t.Fatal("download-only job not complete", got)
			}
			assertDirectOnly(t, c)
		})
	}
}

func TestDirectTorrentPreservesStructureAndAttachments(t *testing.T) {
	w, c, j, _ := directFixture(t)
	j.FileID = "pack"
	c.Files["pack"] = pikpak.File{ID: "pack", ParentID: "dest", Name: "pack (1)", Kind: "drive#folder", Phase: "PHASE_TYPE_COMPLETE"}
	c.Files["nested"] = pikpak.File{ID: "nested", ParentID: "pack", Name: "nested", Kind: "drive#folder", Phase: "PHASE_TYPE_COMPLETE"}
	c.Files["media"] = pikpak.File{ID: "media", ParentID: "nested", Name: "old.mkv", Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
	c.Files["other"] = pikpak.File{ID: "other", ParentID: "pack", Name: "unchanged.mp4", Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
	c.Files["attachment"] = pikpak.File{ID: "attachment", ParentID: "nested", Name: "old.srt", Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
	c.Files["unrelated"] = pikpak.File{ID: "unrelated", ParentID: "dest", Name: "old.mkv", Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
	ctx := context.Background()
	if err := w.DB.SaveJob(ctx, &j); err != nil {
		t.Fatal(err)
	}
	if err := w.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if got := saved(t, w, j); got.State != "complete" || c.Files["media"].Name != "new.mkv" || c.Files["media"].ParentID != "nested" || c.Files["attachment"].Name != "old.srt" || c.Files["unrelated"].Name != "old.mkv" || c.Calls["rename"] != 1 {
		t.Fatal("torrent structure or unrelated files changed", got)
	}
	assertDirectOnly(t, c)
}

type directAPIProvider struct{ api pikpak.API }

func (p directAPIProvider) Snapshot() (pikpak.API, string) { return p.api, "test-account" }
func (p directAPIProvider) Pause(*pikpak.Error)            {}

type renameBoundary struct {
	*testutil.Cloud
	suffix, lost, rejected, readLost bool
}

type secondRenameFailure struct{ *testutil.Cloud }

func (c secondRenameFailure) Rename(ctx context.Context, id, name string) error {
	if id == "second" {
		return &pikpak.Error{Kind: "permanent", Message: "rename rejected"}
	}
	return c.Cloud.Rename(ctx, id, name)
}

func TestDirectPartialRenamesResumeWithoutRepeating(t *testing.T) {
	w, c, j, _ := directFixture(t)
	j.FileID = "pack"
	c.Files["pack"] = pikpak.File{ID: "pack", ParentID: "dest", Name: "pack", Kind: "drive#folder", Phase: "PHASE_TYPE_COMPLETE"}
	c.Files["media"] = pikpak.File{ID: "media", ParentID: "pack", Name: "old.mkv", Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
	c.Files["second"] = pikpak.File{ID: "second", ParentID: "pack", Name: "old.mp4", Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
	w.Provider = directAPIProvider{secondRenameFailure{c}}
	ctx := context.Background()
	if err := w.DB.SaveJob(ctx, &j); err != nil {
		t.Fatal(err)
	}
	if err := w.Process(ctx, j.ID); err == nil {
		t.Fatal("expected partial failure")
	}
	if err := w.Retry(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if err := w.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if c.Calls["rename"] != 1 || saved(t, w, j).State != "needs_review" {
		t.Fatal("successful rename was repeated")
	}
	w.Provider = directAPIProvider{c}
	if err := w.Retry(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if err := w.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if c.Calls["rename"] != 2 || saved(t, w, j).State != "complete" {
		t.Fatal("partial job did not resume")
	}
	assertDirectOnly(t, c)
}

func TestDirectSubmissionAuthQuotaAndRate(t *testing.T) {
	for _, kind := range []string{"auth", "quota", "rate"} {
		w, c, p, _, sub, _ := setup(t)
		j := enqueue(t, w, sub, "queued")
		j.DownloadMode, j.Destination, j.DestinationID = "direct", "", ""
		c.Fail["submit"] = &pikpak.Error{Kind: kind, Message: "fixture failure"}
		ctx := context.Background()
		if err := w.DB.SaveJob(ctx, &j); err != nil {
			t.Fatal(err)
		}
		if err := w.Process(ctx, j.ID); err == nil {
			t.Fatal("expected submission failure")
		}
		got := saved(t, w, j)
		if kind == "rate" {
			if got.State != "queued" || got.NextAttempt == 0 {
				t.Fatal("rate limit not delayed")
			}
		} else if got.State != "paused_"+kind || p.paused == nil {
			t.Fatal("account not paused", got.State)
		}
		assertDirectOnly(t, c)
	}
}
func (c renameBoundary) Rename(ctx context.Context, id, name string) error {
	if c.rejected {
		return &pikpak.Error{Kind: "permanent", Message: "rename rejected"}
	}
	if c.suffix {
		name = "new (1).mkv"
	}
	if err := c.Cloud.Rename(ctx, id, name); err != nil {
		return err
	}
	if c.readLost {
		c.Fail["get"] = errors.New("read response lost")
	}
	if c.lost {
		return errors.New("rename response lost")
	}
	return nil
}

func TestDirectRenameSuffixAndRestartRecovery(t *testing.T) {
	for _, boundary := range []string{"suffix", "lost rename", "lost read", "rejected"} {
		t.Run(boundary, func(t *testing.T) {
			w, c, j, dir := directFixture(t)
			api := renameBoundary{Cloud: c, suffix: true, lost: boundary == "lost rename", readLost: boundary == "lost read", rejected: boundary == "rejected"}
			w.Provider = directAPIProvider{api}
			ctx := context.Background()
			err := w.Process(ctx, j.ID)
			if boundary == "suffix" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("failure was not returned")
			}
			if err := w.DB.Close(); err != nil {
				t.Fatal(err)
			}
			db, err := store.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			w = New(db, directAPIProvider{api}, &feeds{})
			if boundary == "rejected" {
				if err := w.Retry(ctx, j.ID); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.Process(ctx, j.ID); err != nil {
				t.Fatal(err)
			}
			actions, err := db.Actions(ctx, j.ID)
			if err != nil || len(actions) != 1 {
				t.Fatal("missing durable action", err)
			}
			if boundary == "rejected" {
				if got := saved(t, w, j); got.State != "needs_review" || c.Files["media"].Name != "old.mkv" || actions[0].State != "review" {
					t.Fatal("rejected rename changed content", got)
				}
			} else if got := saved(t, w, j); got.State != "complete" || actions[0].ActualName != "new (1).mkv" || actions[0].OriginalName != "old.mkv" || c.Calls["rename"] != 1 {
				t.Fatal("rename was repeated or suffix lost", got, actions)
			}
			assertDirectOnly(t, c)
		})
	}
}

func TestDirectUnknownSubmissionNeverScansOrResubmits(t *testing.T) {
	w, c, j, _ := directFixture(t)
	j.State, j.FileID = "queued", ""
	c.Fail["submit"] = errors.New("lost response")
	ctx := context.Background()
	if err := w.DB.SaveJob(ctx, &j); err != nil {
		t.Fatal(err)
	}
	if err := w.Process(ctx, j.ID); err == nil {
		t.Fatal("expected unknown submission")
	}
	for range 2 {
		if err := w.Retry(ctx, j.ID); err != nil {
			t.Fatal(err)
		}
		if err := w.Process(ctx, j.ID); err != nil {
			t.Fatal(err)
		}
	}
	if got := saved(t, w, j); got.State != "submission_unknown" || c.Calls["submit"] != 1 || c.Calls["list"] != 0 || c.Calls["rename"] != 0 {
		t.Fatal("unknown submission guessed or repeated", got)
	}
	assertDirectOnly(t, c)
}

func TestDirectCompletionWithoutFileID(t *testing.T) {
	for _, renaming := range []bool{false, true} {
		w, c, j, _ := directFixture(t)
		j.Rule.RenameEnabled, j.State, j.TaskID, j.FileID = &renaming, "downloading", "task", ""
		c.Tasks["task"] = pikpak.Task{ID: "task", Status: "complete"}
		ctx := context.Background()
		if err := w.DB.SaveJob(ctx, &j); err != nil {
			t.Fatal(err)
		}
		if err := w.Process(ctx, j.ID); err != nil {
			t.Fatal(err)
		}
		want := "complete"
		if renaming {
			want = "needs_review"
		}
		if got := saved(t, w, j); got.State != want || c.Calls["list"] != 0 || c.Calls["rename"] != 0 {
			t.Fatal("completion guessed a file", got)
		}
	}
}

func TestDirectMissingTaskUsesOnlyKnownFileID(t *testing.T) {
	for _, id := range []string{"", "media"} {
		w, c, j, _ := directFixture(t)
		j.State, j.TaskID, j.FileID = "downloading", "missing", id
		c.Fail["task"] = &pikpak.Error{Kind: "not_found", Message: "missing task"}
		ctx := context.Background()
		if err := w.DB.SaveJob(ctx, &j); err != nil {
			t.Fatal(err)
		}
		if err := w.Process(ctx, j.ID); err != nil {
			t.Fatal(err)
		}
		want := "needs_review"
		if id != "" {
			want = "organizing"
		}
		if got := saved(t, w, j); got.State != want || c.Calls["list"] != 0 || c.Calls["submit"] != 0 {
			t.Fatal("missing task scanned destination", got)
		}
	}
}

func TestDirectRetryWaitsForDownloadBeforeRenaming(t *testing.T) {
	w, c, j, _ := directFixture(t)
	j.State, j.TaskID, j.Progress = "paused_auth", "task", 0
	c.Tasks["task"] = pikpak.Task{ID: "task", FileID: "media", Status: "running"}
	f := c.Files["media"]
	f.Phase = "PHASE_TYPE_RUNNING"
	c.Files["media"] = f
	ctx := context.Background()
	if err := w.DB.SaveJob(ctx, &j); err != nil {
		t.Fatal(err)
	}
	if err := w.Retry(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if err := w.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if got := saved(t, w, j); got.State != "downloading" || c.Calls["rename"] != 0 || c.Calls["submit"] != 0 {
		t.Fatal("retry renamed an unfinished download", got)
	}
	// A persisted organizing job must also wait for a partially downloaded root.
	j.State, j.Progress = "organizing", 100
	if err := w.DB.SaveJob(ctx, &j); err != nil {
		t.Fatal(err)
	}
	if err := w.Process(ctx, j.ID); err == nil {
		t.Fatal("incomplete root was accepted")
	}
	if c.Calls["rename"] != 0 {
		t.Fatal("incomplete root was renamed")
	}
}

func TestRSSCreatesDirectJobs(t *testing.T) {
	w, _, _, f, sub, _ := setup(t)
	f.items = []feed.Item{{Fingerprint: "fixture", Title: "fixture", URL: "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567"}}
	if err := w.Check(context.Background(), sub.ID, true); err != nil {
		t.Fatal(err)
	}
	jobs, err := w.DB.Jobs(context.Background(), 10)
	if err != nil || len(jobs) != 1 || jobs[0].DownloadMode != "direct" || jobs[0].Overwrite {
		t.Fatal("RSS job not direct", err)
	}
}
