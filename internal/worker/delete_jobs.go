package worker

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
)

// Serialize removal with submissions, organization and recovery so a deleted
// job cannot continue making cloud changes after the response is returned.
func (w *Worker) DeleteJob(ctx context.Context, id string, localOnly bool) (int64, error) {
	w.Gate.Lock()
	defer w.Gate.Unlock()
	j, err := w.DB.Job(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if j.State != "complete" {
		if !localOnly && j.TaskID == "" && (j.SubmissionPending || j.State == "submitting" || j.State == "submission_unknown") {
			return 0, errors.New("尚無 PikPak 任務 ID，請先在 PikPak 取消後刪除本機紀錄")
		}
		if j.TaskID != "" {
			api, account := w.Provider.Snapshot()
			if api == nil {
				return 0, errors.New("請先重新檢查 PikPak 連線")
			}
			if account != j.AccountID {
				return 0, errors.New("此任務屬於其他 PikPak 帳號")
			}
			_, err := api.Task(ctx, j.TaskID)
			if err != nil && pikpak.Classify(err).Kind != "not_found" {
				return 0, err
			}
			if err == nil {
				cancel, ok := api.(pikpak.TaskCanceller)
				if !ok {
					return 0, errors.New("官方 MCP 缺少必要工具 task_rm")
				}
				if err := cancel.CancelTask(ctx, j.TaskID); err != nil && pikpak.Classify(err).Kind != "not_found" {
					return 0, err
				}
			}
		}
	}
	return w.DB.DeleteJob(ctx, id)
}

func (w *Worker) ClearCompletedJobs(ctx context.Context) (int64, error) {
	w.Gate.Lock()
	defer w.Gate.Unlock()
	return w.DB.ClearCompletedJobs(ctx)
}
