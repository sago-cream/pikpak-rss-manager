package worker

import (
	"context"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

func TestManualJobRestartAndOriginalFilename(t *testing.T) {
	w, c, p, _, _, dir := setup(t)
	ctx := context.Background()
	off := false
	resource, err := feed.NormalizeMagnet("magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567")
	if err != nil {
		t.Fatal(err)
	}
	j, added, err := w.EnqueueManual(ctx, c.AccountID, resource, model.Subscription{Name: "Manual fixture", Destination: "Downloads", RenameEnabled: &off, RenameMode: "replace"})
	if err != nil || !added {
		t.Fatal("queue manual job", err)
	}
	if err := w.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	j, err = w.DB.Job(ctx, j.ID)
	if err != nil || j.TaskID == "" || j.StagingID == "" {
		t.Fatal("task intent not persisted", err)
	}
	stagingParent := c.Files[c.Files[j.StagingID].ParentID]
	if stagingParent.ParentID != j.DestinationID || stagingParent.Name != "_PikPak-RSS-Staging" {
		t.Fatal("manual staging outside destination")
	}
	if err := w.DB.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restarted := New(db, p, &feeds{})
	c.Files["original"] = pikpak.File{ID: "original", Name: "original.mp4", ParentID: j.StagingID, Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
	c.Tasks[j.TaskID] = pikpak.Task{ID: j.TaskID, Status: "complete", FileID: "original"}
	if err := restarted.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	completed, err := db.Job(ctx, j.ID)
	if err != nil || completed.State != "complete" || completed.SubscriptionID != 0 || completed.StagingID != j.StagingID || c.Calls["submit"] != 1 || c.Calls["rename"] != 0 || c.Files["original"].ParentID != j.DestinationID {
		t.Fatal("manual job did not resume safely", err)
	}
	if _, added, err := restarted.EnqueueManual(ctx, c.AccountID, resource, model.Subscription{Name: "Duplicate", RenameEnabled: &off}); err != nil || added {
		t.Fatal("restart lost deduplication", err)
	}
}
