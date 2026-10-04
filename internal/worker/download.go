package worker

import (
	"context"

	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
)

// A task can disappear after completion or cancellation. Recover only from
// its saved official file ID; never scan the destination or resubmit.
func (w *Worker) recoverMissingTask(ctx context.Context, api pikpak.API, j *model.Job) error {
	if j.FileID != "" {
		f, err := api.Get(ctx, j.FileID)
		if err != nil {
			return w.failure(ctx, j, err, false)
		}
		state := (pikpak.Task{Phase: f.Phase}).State()
		if f.ID == j.FileID && !f.Trashed && f.ParentID == j.DestinationID && (state == "complete" || state == "completed") {
			j.Progress, j.Attempts, j.NextAttempt, j.Error = 100, 0, 0, ""
			if !j.Rule.Renaming() {
				return w.completeDirect(ctx, j)
			}
			j.State = "organizing"
			return w.DB.SaveJob(ctx, j)
		}
	}
	return w.reviewDirect(ctx, j, "PikPak 任務不存在，無法確認下載結果；請至 PikPak 核對")
}
