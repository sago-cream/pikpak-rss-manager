package worker

import (
	"context"
	"testing"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/feed"
)

type blockedFeed struct{ started, release chan struct{} }

func (f *blockedFeed) Fetch(ctx context.Context, _ string) ([]feed.Item, error) {
	close(f.started)
	select {
	case <-f.release:
		return nil, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (*blockedFeed) Resolve(context.Context, string) (feed.Resource, error) {
	panic("empty fixture feed cannot resolve")
}

func TestFeedPreferenceChangeWaitsForCurrentCheck(t *testing.T) {
	w, _, _, replacement, sub, _ := setup(t)
	blocked := &blockedFeed{started: make(chan struct{}), release: make(chan struct{})}
	w.SetFeeds(blocked)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	checkDone := make(chan error, 1)
	go func() { checkDone <- w.Check(ctx, sub.ID, false) }()
	select {
	case <-blocked.started:
	case <-ctx.Done():
		t.Fatal("RSS check did not start")
	}
	changed := make(chan struct{})
	go func() { w.SetFeeds(replacement); close(changed) }()
	select {
	case <-changed:
		t.Fatal("preference changed during active check")
	default:
	}
	close(blocked.release)
	if err := <-checkDone; err != nil {
		t.Fatal(err)
	}
	select {
	case <-changed:
	case <-ctx.Done():
		t.Fatal("preference change did not finish")
	}
	if err := w.Check(ctx, sub.ID, false); err != nil {
		t.Fatal("next check did not use replacement client", err)
	}
}
