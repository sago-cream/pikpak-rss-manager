package integration

import (
	"context"
	"testing"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/worker"
)

const fixtureBase = "https://raw.githubusercontent.com/webtorrent/webtorrent-fixtures/bab825076af799eeb37a729de06f2e58bd02557a/fixtures/"

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
