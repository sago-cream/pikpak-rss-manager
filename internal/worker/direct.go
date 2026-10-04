package worker

import (
	"context"

	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/rename"
)

func (w *Worker) completeDirect(ctx context.Context, j *model.Job) error {
	j.State, j.Progress, j.Error = "complete", 100, ""
	j.Attempts, j.NextAttempt = 0, 0
	if err := w.DB.SaveJob(ctx, j); err != nil {
		return err
	}
	return w.DB.Event(ctx, j.ID, j.SubscriptionID, "info", "下載與改名完成")
}

func (w *Worker) reviewDirect(ctx context.Context, j *model.Job, message string) error {
	j.State, j.Error, j.NextAttempt = "needs_review", message, 0
	if err := w.DB.SaveJob(ctx, j); err != nil {
		return err
	}
	return w.DB.Event(ctx, j.ID, j.SubscriptionID, "warning", message)
}

// Direct jobs operate only on the root ID returned by PikPak and its children.
// No destination scan, move, backup, overwrite or cloud cleanup is performed.
func (w *Worker) renameDirect(ctx context.Context, api pikpak.API, j *model.Job) error {
	if !j.Rule.Renaming() {
		return w.completeDirect(ctx, j)
	}
	if j.FileID == "" {
		return w.reviewDirect(ctx, j, "下載已完成，但沒有可靠的檔案 ID，無法改名；請至 PikPak 核對")
	}
	root, err := api.Get(ctx, j.FileID)
	if err != nil {
		return err
	}
	if root.ID != j.FileID || root.Trashed || root.ParentID != j.DestinationID {
		return w.reviewDirect(ctx, j, "下載檔案的位置已變更，請至 PikPak 核對")
	}
	rootState := (pikpak.Task{Phase: root.Phase}).State()
	if rootState != "complete" && rootState != "completed" {
		return &pikpak.Error{Kind: "transient", Message: "雲端檔案尚未完成，稍後改名"}
	}
	entries := []entry{}
	if err := walk(ctx, api, root, "", 0, &entries); err != nil {
		return err
	}
	if len(entries) == 0 {
		return w.reviewDirect(ctx, j, "完成的任務沒有檔案，請至 PikPak 檢查")
	}
	actions, err := w.DB.Actions(ctx, j.ID)
	if err != nil {
		return err
	}
	known := map[string]model.FileAction{}
	for _, a := range actions {
		known[a.FileID] = a
	}
	if j.FileCount == 0 {
		j.FileCount = len(entries)
		if err := w.DB.SaveJob(ctx, j); err != nil {
			return err
		}
	}
	for _, e := range entries {
		a, exists := known[e.file.ID]
		if exists && a.State != "review" {
			continue
		}
		if !exists {
			a = model.FileAction{JobID: j.ID, FileID: e.file.ID, OriginalName: e.file.Name, ActualName: e.file.Name, RelativePath: e.relative, DestinationID: e.file.ParentID}
		}
		a.State, a.Error, a.TargetName = "pending", "", a.OriginalName
		if primary(a.OriginalName) || j.FileCount == 1 {
			p, renderErr := rename.Render(j.Rule, a.OriginalName)
			if renderErr != nil {
				a.State, a.Error = "review", renderErr.Error()
			} else {
				a.TargetName = p.Name
			}
		}
		if err := w.DB.SaveAction(ctx, a); err != nil {
			return err
		}
	}
	actions, err = w.DB.Actions(ctx, j.ID)
	if err != nil {
		return err
	}
	review := false
	for _, a := range actions {
		if a.State == "done" {
			continue
		}
		if a.State == "review" {
			review = true
			continue
		}
		current, err := api.Get(ctx, a.FileID)
		if err != nil {
			return err
		}
		a.ActualName = current.Name
		state := (pikpak.Task{Phase: current.Phase}).State()
		if current.ID != a.FileID || current.Trashed || current.Folder() || current.ParentID != a.DestinationID {
			a.State, a.Error = "review", "下載檔案的位置已變更，請至 PikPak 核對"
		} else if state != "complete" && state != "completed" {
			return &pikpak.Error{Kind: "transient", Message: "雲端檔案尚未完成，稍後改名"}
		} else if a.State == "renaming" {
			// A lost response may have applied a server-generated suffix. Never
			// repeat an uncertain rename; reconcile by ID and the original name.
			if current.Name == a.OriginalName && current.Name != a.TargetName {
				a.State, a.Error = "review", "改名結果無法確認，已保留目前檔名。PikPak 不允許改成同資料夾內的重複檔名，請檢查是否有同名檔案並排除衝突後重試"
			} else {
				a.State, a.ActualName = "done", current.Name
			}
		} else if current.Name != a.OriginalName && current.Name != a.TargetName {
			a.State, a.Error = "review", "下載檔案的名稱已變更，保留目前檔名；請至 PikPak 核對"
		} else {
			if current.Name != a.TargetName {
				a.State = "renaming"
				if err := w.DB.SaveAction(ctx, a); err != nil {
					return err
				}
				if err := api.Rename(ctx, a.FileID, a.TargetName); err != nil {
					// Keep renaming intent durable until the result is read back.
					return err
				}
				current, err = api.Get(ctx, a.FileID)
				if err != nil {
					return err
				}
				a.ActualName = current.Name
				if current.ID != a.FileID || current.Trashed || current.Folder() || current.ParentID != a.DestinationID || current.Name == a.OriginalName {
					a.State, a.Error = "review", "改名結果無法確認，已保留目前檔名。PikPak 不允許改成同資料夾內的重複檔名，請檢查是否有同名檔案並排除衝突後重試"
				}
			}
			if a.State != "review" {
				a.State, a.ActualName = "done", current.Name
			}
		}
		if a.State == "review" {
			review = true
		}
		if err := w.DB.SaveAction(ctx, a); err != nil {
			return err
		}
	}
	if review {
		return w.reviewDirect(ctx, j, "下載已完成，部分檔案改名待處理；目前檔案已保留")
	}
	return w.completeDirect(ctx, j)
}
