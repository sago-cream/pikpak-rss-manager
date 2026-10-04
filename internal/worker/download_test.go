package worker

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

func TestMissingTaskRecovery(t *testing.T) {
	for _, scenario := range []string{"empty", "no staging id", "wrong staging identity", "trashed staging", "partial", "unknown phase", "trashed root", "multiple roots", "empty root id", "complete file", "complete folder"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			w, c, _, _, sub, dir := setup(t)
			disabled := false
			sub.RenameEnabled = &disabled
			j := enqueue(t, w, sub, "downloading")
			j.TaskID, j.StagingID, j.DestinationID = "missing-task", "stage", "dest"
			c.Files["stage"] = pikpak.File{ID: "stage", Kind: "drive#folder", Name: j.ID}
			c.Files["dest"] = pikpak.File{ID: "dest", Kind: "drive#folder"}
			root := pikpak.File{ID: "root", ParentID: "stage", Kind: "drive#file", Name: "fixture.txt", Phase: "PHASE_TYPE_COMPLETE"}
			if scenario != "empty" {
				c.Files["root"] = root
			}
			switch scenario {
			case "no staging id":
				j.StagingID = ""
				root.ParentID = ""
				c.Files["root"] = root
			case "wrong staging identity":
				c.Files["stage"] = pikpak.File{ID: "another-stage", Kind: "drive#folder"}
			case "trashed staging":
				c.Files["stage"] = pikpak.File{ID: "stage", Kind: "drive#folder", Trashed: true}
			case "partial":
				root.Phase = "PHASE_TYPE_RUNNING"
				c.Files["root"] = root
			case "unknown phase":
				root.Phase = ""
				c.Files["root"] = root
			case "trashed root":
				root.Trashed = true
				c.Files["root"] = root
			case "multiple roots":
				root.ID = "second"
				c.Files["second"] = root
			case "empty root id":
				root.ID = ""
				c.Files["root"] = root
			case "complete folder":
				root.Kind = "drive#folder"
				root.Phase = "Complete (PHASE_TYPE_COMPLETE)"
				c.Files["root"] = root
				c.Files["child"] = pikpak.File{ID: "child", ParentID: "root", Kind: "drive#file", Name: "fixture.txt", Phase: "PHASE_TYPE_COMPLETE"}
			}
			if err := w.DB.SaveJob(ctx, &j); err != nil {
				t.Fatal(err)
			}
			c.Fail["task"] = errors.New("get task failed with status 404")
			if scenario == "empty" {
				c.Fail["get"] = errors.New("get file detail failed with status 400")
			}
			if err := w.Process(ctx, j.ID); err != nil {
				t.Fatal(err)
			}
			got := saved(t, w, j)
			complete := strings.HasPrefix(scenario, "complete")
			if complete {
				if got.State != "organizing" || got.FileID != "root" || got.Progress != 100 || got.Error != "" {
					t.Fatal("completed root was not recovered", got.State)
				}
			} else if got.State != "needs_review" || got.Error == "" || got.FileID != "" {
				t.Fatal("missing task without a unique completed root must require review", got.State)
			}
			if c.Calls["submit"] != 0 || c.Calls["move"] != 0 || c.Calls["rename"] != 0 || c.Calls["trash"] != 0 {
				t.Fatal("missing-task check mutated cloud content")
			}
			if scenario == "no staging id" && (c.Calls["get"] != 0 || c.Calls["list"] != 0) {
				t.Fatal("missing staging ID scanned cloud root")
			}
			if scenario == "empty" && c.Calls["get"] != 0 {
				t.Fatal("empty staging listing queried deleted folder metadata")
			}
			if !complete {
				return
			}
			// Recovery is durable: resume organization after reopening SQLite.
			if err := w.DB.Close(); err != nil {
				t.Fatal(err)
			}
			db, err := store.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			w.DB = db
			if err := w.Process(ctx, j.ID); err != nil {
				t.Fatal(err)
			}
			fileID := "root"
			if scenario == "complete folder" {
				fileID = "child"
			}
			if saved(t, w, j).State != "complete" || c.Files[fileID].ParentID != "dest" || c.Files[fileID].Name != "fixture.txt" || c.Calls["submit"] != 0 || c.Calls["task"] != 1 {
				t.Fatal("restart did not finish the recovered download without resubmission")
			}
		})
	}
}

func TestMissingTaskStagingReadFailure(t *testing.T) {
	for _, kind := range []string{"transient", "rate", "auth", "quota", "not_found"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			w, c, p, _, sub, _ := setup(t)
			j := enqueue(t, w, sub, "downloading")
			j.TaskID, j.StagingID = "missing-task", "stage"
			if err := w.DB.SaveJob(ctx, &j); err != nil {
				t.Fatal(err)
			}
			c.Fail["task"] = &pikpak.Error{Kind: "not_found", Message: "missing"}
			c.Files["root"] = pikpak.File{ID: "root", ParentID: "stage", Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
			c.Fail["get"] = &pikpak.Error{Kind: kind, Message: "staging unavailable"}
			if err := w.Process(ctx, j.ID); err == nil {
				t.Fatal("staging read failure lost")
			}
			want := "downloading"
			if kind == "auth" || kind == "quota" {
				want = "paused_" + kind
				if p.paused == nil {
					t.Fatal("account failure did not pause provider")
				}
			} else if kind == "not_found" {
				want = "needs_review"
			}
			if got := saved(t, w, j); got.State != want || got.TaskID != j.TaskID || got.Attempts != 1 || c.Calls["submit"] != 0 {
				t.Fatal("staging failure bypassed normal recovery/backoff", got.State)
			}
		})
	}
}
