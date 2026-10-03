package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

type uncertainBackup struct {
	pikpak.API
	fail bool
	old  string
}

func (c *uncertainBackup) Move(ctx context.Context, id, parent string) error {
	if err := c.API.Move(ctx, id, parent); err != nil {
		return err
	}
	if id == c.old && c.fail {
		c.fail = false
		return errors.New("uncertain move")
	}
	return nil
}

type overwriteProvider struct{ api pikpak.API }

func (p *overwriteProvider) Snapshot() (pikpak.API, string) { return p.api, "test-account" }
func (p *overwriteProvider) Pause(*pikpak.Error)            {}

func TestOverwriteBacksUpOriginalAndRecoversUncertainMove(t *testing.T) {
	w, c, _, _, sub, dir := setup(t)
	ctx := context.Background()
	j := enqueue(t, w, sub, "organizing")
	j.Overwrite = true
	off := false
	j.Rule.RenameEnabled = &off
	prepareFiles(t, w, c, &j, pikpak.File{ID: "new", Name: "Fixture.mkv"})
	c.Files["old"] = pikpak.File{ID: "old", Name: "Fixture.mkv", ParentID: "dest", Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
	api := &uncertainBackup{API: c, fail: true, old: "old"}
	w.Provider = &overwriteProvider{api}
	if err := w.Process(ctx, j.ID); err == nil {
		t.Fatal("uncertain move did not fail")
	}
	actions, err := w.DB.Actions(ctx, j.ID)
	if err != nil || len(actions) != 1 || len(actions[0].ReplacementIDs) != 1 || actions[0].BackupID == "" || c.Files["old"].ParentID != actions[0].BackupID || c.Files["new"].ParentID == "dest" {
		t.Fatal("backup intent not persisted", err)
	}
	if err := w.DB.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restarted := New(db, &overwriteProvider{api}, &feeds{})
	if err := restarted.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	got, err := db.Job(ctx, j.ID)
	if err != nil || got.State != "complete" || !got.Overwrite || c.Files["new"].ParentID != "dest" || c.Files["old"].ParentID != actions[0].BackupID || c.Calls["move"] != 2 {
		t.Fatal("replacement recovery failed", err)
	}
	if err := restarted.Process(ctx, j.ID); err != nil || c.Calls["move"] != 2 {
		t.Fatal("completed replacement repeated")
	}
}

func TestOverwriteDoesNotReplaceFolders(t *testing.T) {
	w, c, _, _, sub, _ := setup(t)
	j := enqueue(t, w, sub, "organizing")
	j.Overwrite = true
	off := false
	j.Rule.RenameEnabled = &off
	prepareFiles(t, w, c, &j, pikpak.File{ID: "new", Name: "Fixture.mkv"})
	c.Files["folder"] = pikpak.File{ID: "folder", Name: "Fixture.mkv", ParentID: "dest", Kind: "drive#folder"}
	if err := w.Process(context.Background(), j.ID); err != nil {
		t.Fatal(err)
	}
	if saved(t, w, j).State != "needs_review" || c.Calls["move"] != 0 {
		t.Fatal("replacement moved a folder")
	}
}
