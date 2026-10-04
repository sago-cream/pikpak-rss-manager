package worker

import (
	"context"

	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
)

// A task can disappear after completion or cancellation. Recover only from a
// uniquely completed root in its persisted staging folder; never resubmit.
func (w *Worker) recoverMissingTask(ctx context.Context, api pikpak.API, j *model.Job) error {
	if j.DownloadMode == "direct" {
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
	if j.StagingID != "" {
		roots, err := pikpak.ListAll(ctx, api, j.StagingID)
		if err != nil {
			return w.failure(ctx, j, err, false)
		}
		if len(roots) == 1 {
			root := roots[0]
			state := (pikpak.Task{Phase: root.Phase}).State()
			if root.ID == "" || root.ParentID != j.StagingID || root.Trashed || (state != "complete" && state != "completed") {
				return w.reviewMissingTask(ctx, j)
			}
			// Validate folder identity only when there is content to recover.
			// PikPak can return 400 for a deleted folder even when ls is empty.
			stage, err := api.Get(ctx, j.StagingID)
			if err != nil {
				return w.failure(ctx, j, err, false)
			}
			if stage.ID == j.StagingID && stage.Folder() && !stage.Trashed {
				j.FileID = root.ID
				j.State = "organizing"
				j.Progress = 100
				j.Error = ""
				j.Attempts = 0
				j.NextAttempt = 0
				return w.DB.SaveJob(ctx, j)
			}
		}
	}
	return w.reviewMissingTask(ctx, j)
}

func (w *Worker) reviewMissingTask(ctx context.Context, j *model.Job) error {
	j.State = "needs_review"
	j.Error = "PikPak 離線任務不存在，無法確認下載結果；請核對 PikPak 任務及暫存目錄"
	j.NextAttempt = 0
	if err := w.DB.SaveJob(ctx, j); err != nil {
		return err
	}
	return w.DB.Event(ctx, j.ID, j.SubscriptionID, "warning", j.Error)
}
