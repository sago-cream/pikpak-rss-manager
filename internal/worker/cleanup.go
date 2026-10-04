package worker

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
)

// Cleanup has its own bounded retry budget. It never resubmits, reorganizes,
// changes a completed job's state or pauses otherwise usable cloud credentials.
func (w *Worker) cleanupCompleted(ctx context.Context, api pikpak.API, j *model.Job) error {
	c := j.StagingCleanup
	if j.State != "complete" || c == nil || !c.Pending {
		return nil
	}
	cleanupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := w.cleanupStaging(cleanupCtx, api, j)
	c.Attempts++
	c.Pending = false
	j.NextAttempt = 0
	if err != nil {
		kind := pikpak.Classify(err).Kind
		if c.Attempts < 3 && (kind == "transient" || kind == "rate") {
			c.Pending = true
			j.NextAttempt = time.Now().Add(time.Duration(c.Attempts) * time.Minute).Unix()
		}
		if c.Attempts == 1 {
			_ = w.DB.Event(ctx, j.ID, j.SubscriptionID, "warning", "下載已完成，但空暫存資料夾清理失敗；請確認 PAT 的管理檔案權限")
		}
	}
	return w.DB.SaveJob(ctx, j)
}

func (w *Worker) cleanupStaging(ctx context.Context, api pikpak.API, j *model.Job) error {
	parent, err := api.Get(ctx, j.StagingCleanup.ParentID)
	if err != nil {
		// rm may have succeeded just before a crash or a lost response. Confirm
		// absence by ID in the destination, without retrying an uncertain rm.
		if w.StagingPath != "" {
			return err
		}
		files, listErr := pikpak.ListAll(ctx, api, j.DestinationID)
		if listErr != nil {
			return listErr
		}
		for _, f := range files {
			if f.ID == j.StagingCleanup.ParentID {
				return err
			}
		}
		return nil
	}
	if parent.ID != j.StagingCleanup.ParentID || !parent.Folder() || parent.Trashed {
		return nil
	}
	if w.StagingPath == "" && (parent.Name != "_PikPak-RSS-Staging" || parent.ParentID != j.DestinationID) {
		return nil
	}
	files, err := pikpak.ListAll(ctx, api, parent.ID)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.ID != j.StagingID {
			continue
		}
		if !f.Folder() || f.Name != j.ID || f.ParentID != parent.ID {
			return nil
		}
		budget := 5000
		if _, err := pruneEmpty(ctx, api, f, 0, &budget); err != nil {
			return err
		}
	}
	// Do not recurse into sibling jobs, legacy staging or test namespaces.
	if w.StagingPath == "" {
		_, err = trashEmpty(ctx, api, parent)
	}
	return err
}

func pruneEmpty(ctx context.Context, api pikpak.API, folder pikpak.File, depth int, budget *int) (bool, error) {
	if depth > 32 || *budget <= 0 {
		return false, &pikpak.Error{Kind: "permanent", Message: "暫存資料夾清理超過安全上限"}
	}
	*budget--
	if !folder.Folder() || folder.Trashed || folder.Name == "_Replaced" || (folder.Phase != "" && !strings.Contains(strings.ToLower(folder.Phase), "complete")) {
		return false, nil
	}
	files, err := pikpak.ListAll(ctx, api, folder.ID)
	if err != nil {
		return false, err
	}
	if len(files) > *budget {
		return false, &pikpak.Error{Kind: "permanent", Message: "暫存資料夾清理超過安全上限"}
	}
	for _, f := range files {
		if f.ParentID != folder.ID || !f.Folder() {
			continue
		}
		if _, err := pruneEmpty(ctx, api, f, depth+1, budget); err != nil {
			return false, err
		}
	}
	return trashEmpty(ctx, api, folder)
}

func trashEmpty(ctx context.Context, api pikpak.API, expected pikpak.File) (bool, error) {
	current, err := api.Get(ctx, expected.ID)
	if err != nil {
		return false, err
	}
	if current.ID != expected.ID || !current.Folder() || current.Trashed || current.ParentID != expected.ParentID || current.Name != expected.Name {
		return false, nil
	}
	// Recheck emptiness immediately before the recoverable mutation. Never
	// remove a folder merely because its earlier snapshot contained no files.
	files, err := pikpak.ListAll(ctx, api, current.ID)
	if err != nil || len(files) != 0 {
		return false, err
	}
	trasher, ok := api.(interface {
		Trash(context.Context, string) error
	})
	if !ok {
		return false, errors.New("cloud trash operation unavailable")
	}
	if err := trasher.Trash(ctx, current.ID); err != nil {
		return false, err
	}
	return true, nil
}
