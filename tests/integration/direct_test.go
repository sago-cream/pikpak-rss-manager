package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/worker"
)

// Only download/rename inside a newly created test run. Never move, trash or
// delete content; preserve all downloaded public-domain fixtures afterward.
func TestLiveDirectDownloadAndRename(t *testing.T) {
	if os.Getenv("PIKPAK_LIVE_DIRECT_TEST") != "1" {
		t.Skip("opt-in direct cloud test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	api, err := pikpak.New(ctx, liveToken(t))
	if err != nil {
		t.Fatal("official MCP unavailable", err)
	}
	defer api.Close()
	account, err := api.Account(ctx)
	if err != nil {
		t.Fatal("account unavailable", err)
	}
	base, err := pikpak.EnsureFolder(ctx, api, "", "_pikpak-rss-manager-test")
	if err != nil {
		t.Fatal("test namespace unavailable", err)
	}
	name := "direct-" + time.Now().UTC().Format("20060102T150405") + "-" + store.ID()[:6]
	run := "_pikpak-rss-manager-test/" + name
	destination, err := pikpak.EnsureFolder(ctx, api, base, name)
	if err != nil {
		t.Fatal("test destination unavailable", err)
	}
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	w := worker.New(db, &liveProvider{api: api, account: account.ID}, feed.New(false))
	off, on := false, true
	var files []pikpak.File
	for i := 0; i < 3; i++ {
		sub := model.Subscription{Name: "Public Domain Alice", Destination: run, DestinationID: destination, RenameEnabled: off}
		if i == 2 {
			sub.RenameEnabled = on
			sub.Regex, sub.Replacement = `^alice.*\.txt$`, "direct-renamed.txt"
		}
		j, added, err := w.EnqueueManual(ctx, account.ID, feed.Resource{Key: "direct:" + store.ID(), URL: fixtureBase + "alice.txt"}, sub)
		if err != nil || !added {
			t.Fatal("test queue failed", err)
		}
		if !finishLive(ctx, t, w, db, j.ID, 90*time.Second) {
			t.Fatal("direct fixture did not finish; preserved run", run)
		}
		j, err = db.Job(ctx, j.ID)
		if err != nil || j.FileID == "" {
			t.Fatal("direct job or reliable file ID not verified", err)
		}
		f, err := api.Get(ctx, j.FileID)
		if err != nil || f.ID != j.FileID || f.ParentID != destination || f.Folder() || f.Trashed || f.Size != "163783" {
			t.Fatal("direct fixture identity not verified", err)
		}
		files = append(files, f)
		if i == 2 {
			actions, err := db.Actions(ctx, j.ID)
			if err != nil || len(actions) != 1 || actions[0].State != "done" || actions[0].ActualName != f.Name || f.Name != "direct-renamed.txt" {
				t.Fatal("direct rename not verified", err)
			}
		}
	}
	if files[0].ID == files[1].ID || files[0].Name == files[1].Name {
		t.Fatal("repeat download did not create distinct filenames")
	}
	first, err := api.Get(ctx, files[0].ID)
	if err != nil || first.Name != files[0].Name || first.ParentID != destination {
		t.Fatal("repeat download modified original", err)
	}
	t.Log("Verified direct HTTP download, automatic duplicate-download filenames and worker rename; preserved run:", run)
	t.Logf("Duplicate-download filenames: %q and %q", files[0].Name, files[1].Name)
	// Probe rename collisions separately; download numbering does not prove
	// rename numbering. A rejection is valid if the current file is retained.
	renameErr := api.Rename(ctx, files[2].ID, files[1].Name)
	renamed, err := api.Get(ctx, files[2].ID)
	if err != nil || renamed.ParentID != destination || renamed.Trashed || renamed.Size != "163783" {
		t.Fatal("rename collision lost fixture", err)
	}
	if renameErr != nil {
		if renamed.Name != files[2].Name {
			t.Fatal("rejected rename unexpectedly changed filename")
		}
		t.Log("Rename collision rejected; current filename retained:", pikpak.Classify(renameErr).Message)
	} else {
		t.Logf("Rename collision accepted; requested=%q actual=%q", files[1].Name, renamed.Name)
	}
	for _, expected := range files[:2] {
		f, err := api.Get(ctx, expected.ID)
		if err != nil || f.Name != expected.Name || f.Size != expected.Size || f.ParentID != destination || f.Trashed {
			t.Fatal("rename collision changed another fixture", err)
		}
	}
	listing, err := pikpak.ListAll(ctx, api, destination)
	if err != nil || len(listing) != 3 {
		t.Fatal("unexpected direct destination content", err)
	}
	for _, f := range listing {
		if f.Folder() {
			t.Fatal("direct download created an extra folder")
		}
	}
}
