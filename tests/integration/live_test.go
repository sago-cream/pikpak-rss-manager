package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/worker"
)

const fixtureBase = "https://raw.githubusercontent.com/webtorrent/webtorrent-fixtures/bab825076af799eeb37a729de06f2e58bd02557a/fixtures/"

// These tests never run in CI. They intentionally preserve the created cloud
// content. All mutations are confined to a new dedicated test directory.
func TestLivePikPak(t *testing.T) {
	if os.Getenv("PIKPAK_LIVE_TEST") != "1" {
		t.Skip("opt-in local cloud test")
	}
	_ = godotenv.Load(filepath.Join("..", "..", ".env"))
	token := os.Getenv("PIKPAK_TOKEN")
	if token == "" {
		t.Fatal("local PIKPAK_TOKEN is required; never pass it on the command line")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client, err := pikpak.New(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	account, err := client.Account(ctx)
	if err != nil {
		t.Fatal(err)
	}
	run := "run-" + time.Now().UTC().Format("20060102T150405") + "-" + store.ID()[:6]
	cloudRoot := "_pikpak-rss-manager-test/" + run
	db, err := store.Open(filepath.Join("..", "..", ".local", "live", run))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	provider := &liveProvider{api: client, account: account.ID}
	w := worker.New(db, provider, feed.New(false))
	w.StagingPath = cloudRoot + "/staging"
	sub := model.Subscription{Name: "Public Domain Alice", RSSURL: "https://example.org/test-feed.xml", Destination: cloudRoot + "/organized", Enabled: true, IntervalMinutes: 10, Season: 1, Regex: `S(?P<season>\d+)E(?P<ep>\d+)`, Template: `{title} - S{season:02}E{ep:02}.{ext}`}
	if err := w.SaveSubscription(ctx, &sub); err != nil {
		t.Fatal(err)
	}
	// Only torrent metadata is fetched locally, never the resource contents.
	resource, err := feed.New(false).Resolve(ctx, fixtureBase+"alice.torrent")
	if err != nil {
		t.Fatal(err)
	}
	job := model.Job{ID: store.ID(), SubscriptionID: sub.ID, AccountID: account.ID, ResourceKey: resource.Key, ResourceURL: resource.URL, Title: "Alice S01E01", Rule: sub.Rule(), Destination: sub.Destination, State: "queued", CreatedAt: time.Now().Unix()}
	added, err := db.Enqueue(ctx, job, "magnet-fixture")
	if err != nil || !added {
		t.Fatalf("enqueue: %v", err)
	}
	duplicate := job
	duplicate.ID = store.ID()
	added, err = db.Enqueue(ctx, duplicate, "same-infohash-other-feed-item")
	if err != nil || added {
		t.Fatal("same-account infohash deduplication failed")
	}
	t.Log("Account read succeeded; scoped test root:", cloudRoot)
	magnetComplete := finishLive(ctx, t, w, db, job.ID, 90*time.Second)
	transport := "magnet"
	if !magnetComplete {
		t.Log("Tiny torrent not completed within 90 seconds; preserving task and testing cloud HTTP download with the same public-domain text")
		if provider.paused {
			t.Fatal("authorization or quota paused the test")
		}
		job.ID = store.ID()
		job.ResourceKey = "live-http:" + run
		job.ResourceURL = fixtureBase + "alice.txt"
		job.Title = "Alice S01E02"
		job.State = "queued"
		if _, err := db.Enqueue(ctx, job, "http-fixture"); err != nil {
			t.Fatal(err)
		}
		if !finishLive(ctx, t, w, db, job.ID, 90*time.Second) {
			t.Fatal("cloud HTTP roundtrip did not complete")
		}
		transport = "http (magnet remained pending)"
	}
	finished, err := db.Job(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := db.Actions(ctx, job.ID)
	if err != nil || len(actions) != 1 {
		t.Fatal("expected one durable file action")
	}
	file, err := client.Get(ctx, actions[0].FileID)
	if err != nil {
		t.Fatal(err)
	}
	if actions[0].State != "done" || file.Name != actions[0].TargetName || file.ParentID != finished.DestinationID {
		t.Fatal("cloud rename/move verification failed")
	}
	if file.Size != "163783" {
		t.Fatal("unexpected test resource size; refusing further mutations")
	}
	t.Log("Verified cloud completion, real filename extension, rename, move and duplicate-source suppression using", transport)
	// Only non-sensitive aggregate facts are recorded for the delivery report.
	report := map[string]any{"tested_at": time.Now().UTC().Format(time.RFC3339), "cloud_root": cloudRoot, "transport": transport, "magnet_complete": magnetComplete, "rename_move_verified": true, "dedup_verified": true, "bytes": 163783}
	b, _ := json.MarshalIndent(report, "", "  ")
	reportDir := filepath.Join("..", "..", ".local")
	_ = os.MkdirAll(reportDir, 0700)
	if err := os.WriteFile(filepath.Join(reportDir, "live-test.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
}

type liveProvider struct {
	api     pikpak.API
	account string
	paused  bool
}

func (p *liveProvider) Snapshot() (pikpak.API, string) {
	if p.paused {
		return nil, p.account
	}
	return p.api, p.account
}
func (p *liveProvider) Pause(*pikpak.Error) { p.paused = true }
func finishLive(ctx context.Context, t *testing.T, w *worker.Worker, db *store.Store, id string, limit time.Duration) bool {
	t.Helper()
	until := time.Now().Add(limit)
	for time.Now().Before(until) {
		if err := w.Process(ctx, id); err != nil {
			t.Log("Safe provider error:", err)
		}
		job, err := db.Job(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Cloud workflow state=%s progress=%.0f%%", job.State, job.Progress)
		if job.State == "complete" {
			return true
		}
		if job.State != "queued" && job.State != "downloading" && job.State != "organizing" {
			t.Fatalf("cloud workflow stopped: %s (%s)", job.State, job.Error)
		}
		select {
		case <-ctx.Done():
			t.Fatal("test deadline reached")
		case <-time.After(5 * time.Second):
		}
	}
	return false
}
