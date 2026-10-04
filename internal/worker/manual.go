package worker

import (
	"context"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

// EnqueueManual persists a one-off job without a subscription or feed baseline.
// The caller holds Gate so account/destination validation and insertion are atomic
// with respect to account switches. Process handles all subsequent cloud actions.
func (w *Worker) EnqueueManual(ctx context.Context, account string, resource feed.Resource, sub model.Subscription) (model.Job, bool, error) {
	now := time.Now().Unix()
	j := model.Job{DownloadMode: "direct", ID: store.ID(), AccountID: account, ResourceKey: resource.Key, ResourceURL: resource.URL, Title: sub.Name, Rule: sub.Rule(), Destination: sub.Destination, DestinationID: sub.DestinationID, State: "queued", NextAttempt: now, CreatedAt: now, UpdatedAt: now}
	added, err := w.DB.Enqueue(ctx, j, "")
	if err == nil && added {
		_ = w.DB.Event(ctx, j.ID, 0, "info", "手動任務已排入佇列")
	}
	return j, added, err
}
