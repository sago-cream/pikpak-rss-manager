package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/testutil"
)

func cleanupFixture(t *testing.T) (*Worker, *testutil.Cloud, model.Job, string) {
	t.Helper()
	w, c, _, _, sub, dir := setup(t)
	off := false
	sub.RenameEnabled = &off
	j := enqueue(t, w, sub, "queued")
	if err := w.Process(context.Background(), j.ID); err != nil {
		t.Fatal(err)
	}
	j = saved(t, w, j)
	c.Files["pack"] = pikpak.File{ID: "pack", ParentID: j.StagingID, Name: "pack", Kind: "drive#folder", Phase: "PHASE_TYPE_COMPLETE"}
	c.Files["nested"] = pikpak.File{ID: "nested", ParentID: "pack", Name: "nested", Kind: "drive#folder", Phase: "PHASE_TYPE_COMPLETE"}
	c.Files["media"] = pikpak.File{ID: "media", ParentID: "nested", Name: "fixture.mp4", Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
	c.Tasks[j.TaskID] = pikpak.Task{ID: j.TaskID, FileID: "pack", Status: "complete"}
	return w, c, j, dir
}

func TestCompletedStagingCleanup(t *testing.T) {
	w, c, j, _ := cleanupFixture(t)
	parent := j.StagingCleanup.ParentID
	if err := w.Process(context.Background(), j.ID); err != nil {
		t.Fatal(err)
	}
	got := saved(t, w, j)
	if got.State != "complete" || got.StagingCleanup.Pending || c.Calls["trash"] != 4 || c.Calls["submit"] != 1 {
		t.Fatal("cleanup did not finish independently", got.State, c.Calls)
	}
	for _, id := range []string{"pack", "nested", j.StagingID, parent} {
		if _, ok := c.Files[id]; ok {
			t.Fatal("empty folder retained", id)
		}
	}
	if c.Files["media"].ParentID != j.DestinationID {
		t.Fatal("downloaded file lost")
	}
	if _, ok := c.Files[j.DestinationID]; !ok {
		t.Fatal("destination removed")
	}
}

func TestCleanupPreservesBackupsFilesAndOtherJobs(t *testing.T) {
	for _, kind := range []string{"backup", "empty backup", "unexpected file", "other job", "legacy job", "review"} {
		t.Run(kind, func(t *testing.T) {
			w, c, j, _ := cleanupFixture(t)
			parent := j.StagingCleanup.ParentID
			switch kind {
			case "backup", "empty backup":
				c.Files["backup"] = pikpak.File{ID: "backup", ParentID: j.StagingID, Name: "_Replaced", Kind: "drive#folder"}
				if kind == "backup" {
					c.Files["old"] = pikpak.File{ID: "old", ParentID: "backup", Name: "old.mp4", Kind: "drive#file"}
				}
			case "unexpected file":
				c.Files["extra"] = pikpak.File{ID: "extra", ParentID: j.StagingID, Name: "keep.txt", Kind: "drive#file"}
			case "other job":
				c.Files["other"] = pikpak.File{ID: "other", ParentID: parent, Name: "unfinished-job", Kind: "drive#folder"}
			case "legacy job":
				j.StagingCleanup = nil
				if err := w.DB.SaveJob(context.Background(), &j); err != nil {
					t.Fatal(err)
				}
			case "review":
				c.Files["collision"] = pikpak.File{ID: "collision", ParentID: j.DestinationID, Name: "fixture.mp4", Kind: "drive#file"}
			}
			if err := w.Process(context.Background(), j.ID); err != nil {
				t.Fatal(err)
			}
			if _, ok := c.Files[parent]; !ok {
				t.Fatal("protected parent removed")
			}
			if kind != "other job" {
				if _, ok := c.Files[j.StagingID]; !ok {
					t.Fatal("protected staging removed")
				}
			}
			if kind == "legacy job" || kind == "review" {
				if c.Calls["trash"] != 0 {
					t.Fatal("unfinished or legacy staging cleaned")
				}
			}
		})
	}
}

func TestCleanupFailureRestartAndBoundedRetries(t *testing.T) {
	w, c, j, dir := cleanupFixture(t)
	c.Fail["trash"] = errors.New("lost response")
	if err := w.Process(context.Background(), j.ID); err != nil {
		t.Fatal(err)
	}
	j = saved(t, w, j)
	if j.State != "complete" || !j.StagingCleanup.Pending || j.StagingCleanup.Attempts != 1 {
		t.Fatal("cleanup changed completion or lost retry")
	}
	if err := w.DB.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restarted := New(db, w.Provider, &feeds{})
	due, err := db.DueJobs(context.Background(), c.AccountID, time.Now().Add(5*time.Minute).Unix())
	if err != nil || len(due) != 1 {
		t.Fatal("completed cleanup not scheduled after restart", err)
	}
	if err := restarted.Process(context.Background(), j.ID); err != nil {
		t.Fatal(err)
	}
	j = saved(t, restarted, j)
	if j.State != "complete" || j.StagingCleanup.Pending || c.Calls["submit"] != 1 || c.Calls["move"] != 1 {
		t.Fatal("cleanup repeated file actions")
	}

	w, c, j, _ = cleanupFixture(t)
	for i := 0; i < 3; i++ {
		c.Fail["trash"] = errors.New("transient")
		if err := w.Process(context.Background(), j.ID); err != nil {
			t.Fatal(err)
		}
	}
	j = saved(t, w, j)
	if j.State != "complete" || j.StagingCleanup.Pending || j.StagingCleanup.Attempts != 3 {
		t.Fatal("unbounded cleanup retries")
	}
}

func TestCleanupAuthDoesNotPauseDownloads(t *testing.T) {
	w, c, j, _ := cleanupFixture(t)
	c.Fail["trash"] = &pikpak.Error{Kind: "auth", Message: "missing manage permission"}
	if err := w.Process(context.Background(), j.ID); err != nil {
		t.Fatal(err)
	}
	j = saved(t, w, j)
	api, _ := w.Provider.Snapshot()
	if j.State != "complete" || j.StagingCleanup.Pending || api == nil || c.Files["media"].ParentID != j.DestinationID {
		t.Fatal("cleanup authorization blocked downloads")
	}
}

type uncertainTrash struct {
	*testutil.Cloud
	loseID string
}

func (c *uncertainTrash) Trash(ctx context.Context, id string) error {
	if err := c.Cloud.Trash(ctx, id); err != nil {
		return err
	}
	if id == c.loseID {
		c.loseID = ""
		return errors.New("response lost after trash")
	}
	return nil
}

func TestCleanupReconcilesUncertainTrashByID(t *testing.T) {
	for _, boundary := range []string{"nested", "stage", "parent"} {
		t.Run(boundary, func(t *testing.T) {
			w, c, j, _ := cleanupFixture(t)
			id := map[string]string{"nested": "nested", "stage": j.StagingID, "parent": j.StagingCleanup.ParentID}[boundary]
			wrapped := &uncertainTrash{Cloud: c, loseID: id}
			j.State = "complete"
			if err := w.DB.SaveJob(context.Background(), &j); err != nil {
				t.Fatal(err)
			}
			// Mimic persisted successful file actions, without organizing again.
			f := c.Files["media"]
			f.ParentID = j.DestinationID
			c.Files["media"] = f
			if err := w.cleanupCompleted(context.Background(), wrapped, &j); err != nil {
				t.Fatal(err)
			}
			if !j.StagingCleanup.Pending {
				t.Fatal("uncertain cleanup not retained")
			}
			if err := w.cleanupCompleted(context.Background(), wrapped, &j); err != nil {
				t.Fatal(err)
			}
			if j.StagingCleanup.Pending || c.Calls["trash"] != 4 {
				t.Fatal("uncertain mutation repeated or unreconciled", c.Calls)
			}
		})
	}
}

type changedCleanupFolder struct {
	*testutil.Cloud
	id string
}

func (c changedCleanupFolder) Get(ctx context.Context, id string) (pikpak.File, error) {
	f, err := c.Cloud.Get(ctx, id)
	if id == c.id {
		f.ParentID = "elsewhere"
	}
	return f, err
}
func TestCleanupRevalidatesFolderAndAccount(t *testing.T) {
	w, c, j, _ := cleanupFixture(t)
	f := c.Files["nested"]
	removed, err := trashEmpty(context.Background(), changedCleanupFolder{c, "nested"}, f)
	if err != nil || removed || c.Calls["trash"] != 0 {
		t.Fatal("moved folder removed")
	}
	j.State = "complete"
	if err := w.DB.SaveJob(context.Background(), &j); err != nil {
		t.Fatal(err)
	}
	c.AccountID = "other-account"
	if err := w.Process(context.Background(), j.ID); err != nil {
		t.Fatal(err)
	}
	if c.Calls["trash"] != 0 || saved(t, w, j).State != "complete" {
		t.Fatal("old account cleanup executed")
	}
}
