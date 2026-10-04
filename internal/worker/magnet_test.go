package worker

import (
	"context"
	"strings"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
)

func TestQueuedLegacyMagnetIsCanonicalizedWithoutResubmission(t *testing.T) {
	ctx := context.Background()
	w, c, _, _, sub, _ := setup(t)
	j := enqueue(t, w, sub, "queued")
	j.ResourceURL = "magnet:?dn=Fixture&xt=urn%3Abtih%3A0123456789ABCDEF0123456789ABCDEF01234567"
	if err := w.DB.SaveJob(ctx, &j); err != nil {
		t.Fatal(err)
	}
	c.OnSubmit = func(parent, source string) (pikpak.Task, error) {
		if !strings.HasPrefix(source, "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&") {
			t.Fatal("legacy queued URL was not canonicalized")
		}
		return pikpak.Task{ID: "task", Status: "running"}, nil
	}
	if err := w.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	c.Tasks["task"] = pikpak.Task{ID: "task", Status: "running", Progress: "0"}
	if err := w.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if got := saved(t, w, j); got.State != "downloading" || got.TaskID != "task" || c.Calls["submit"] != 1 {
		t.Fatal("active pending task was resubmitted")
	}
}
