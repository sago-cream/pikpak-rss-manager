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

// The only cleanup mutations are recoverable rm calls on empty folders created
// in this run. The downloaded public-domain text and run namespace remain.
func TestLiveStagingCleanup(t *testing.T) {
	if os.Getenv("PIKPAK_LIVE_CLEANUP_TEST") != "1" {
		t.Skip("opt-in real cleanup test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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
	run := "_pikpak-rss-manager-test/cleanup-" + time.Now().UTC().Format("20060102T150405") + "-" + store.ID()[:6]
	destination, err := pikpak.EnsureFolder(ctx, api, "", run)
	if err != nil {
		t.Fatal("isolated destination unavailable", err)
	}
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	p := &liveProvider{api: api, account: account.ID}
	w := worker.New(db, p, feed.New(false))
	off := false
	j, added, err := w.EnqueueManual(ctx, account.ID, feed.Resource{Key: "cleanup:" + store.ID(), URL: fixtureBase + "alice.txt"}, model.Subscription{Name: "Public Domain Alice", Destination: run, DestinationID: destination, RenameEnabled: &off})
	if err != nil || !added {
		t.Fatal("isolated task queue failed", err)
	}
	if !finishLive(ctx, t, w, db, j.ID, 90*time.Second) {
		t.Fatal("fixture download did not finish; preserved run", run)
	}
	j, err = db.Job(ctx, j.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Allow bounded cleanup to reconcile an uncertain rm without resubmitting.
	for i := 0; i < 2 && j.StagingCleanup != nil && j.StagingCleanup.Pending; i++ {
		if err := w.Process(ctx, j.ID); err != nil {
			t.Fatal(err)
		}
		j, err = db.Job(ctx, j.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	files, err := pikpak.ListAll(ctx, api, destination)
	if err != nil {
		t.Fatal("destination listing failed", err)
	}
	if len(files) != 1 || files[0].Folder() || files[0].Name != "alice.txt" {
		t.Fatal("completed fixture or empty staging cleanup not verified; inspect scoped run", run)
	}
	if j.StagingCleanup == nil || j.StagingCleanup.Pending {
		t.Fatal("cleanup did not finish")
	}
	t.Log("Verified real HTTP download and recoverable empty staging cleanup; preserved fixture:", run)
}
