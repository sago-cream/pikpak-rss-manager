package worker

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/rename"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/testutil"
)

type provider struct {
	cloud  *testutil.Cloud
	paused *pikpak.Error
}

func (p *provider) Snapshot() (pikpak.API, string) {
	if p.paused != nil {
		return nil, p.cloud.AccountID
	}
	return p.cloud, p.cloud.AccountID
}
func (p *provider) Pause(e *pikpak.Error) { p.paused = e }

type feeds struct {
	items    []feed.Item
	resolves int
}

func (f *feeds) Fetch(context.Context, string) ([]feed.Item, error) { return f.items, nil }
func (f *feeds) Resolve(_ context.Context, v string) (feed.Resource, error) {
	f.resolves++
	return feed.NormalizeMagnet(v)
}

func setup(t *testing.T) (*Worker, *testutil.Cloud, *provider, *feeds, model.Subscription, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "data")
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	cloud := testutil.NewCloud()
	p := &provider{cloud: cloud}
	f := &feeds{}
	w := New(db, p, f)
	sub := model.Subscription{Name: "作品", RSSURL: "https://rss.test", Destination: "Anime/作品", Enabled: true, IntervalMinutes: 10, Season: 1, Regex: rename.DefaultRegex, Template: rename.DefaultTemplate}
	if err := w.SaveSubscription(context.Background(), &sub); err != nil {
		t.Fatal(err)
	}
	return w, cloud, p, f, sub, dir
}
func enqueue(t *testing.T, w *Worker, sub model.Subscription, state string) model.Job {
	t.Helper()
	j := model.Job{ID: store.ID(), SubscriptionID: sub.ID, AccountID: "test-account", ResourceKey: store.ID(), ResourceURL: "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567", Title: "作品 S01E09", Rule: sub.Rule(), Destination: sub.Destination, State: state, CreatedAt: time.Now().Unix()}
	if _, err := w.DB.Enqueue(context.Background(), j, j.ID); err != nil {
		t.Fatal(err)
	}
	return j
}
func saved(t *testing.T, w *Worker, j model.Job) model.Job {
	t.Helper()
	got, err := w.DB.Job(context.Background(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	return got
}
func prepareFiles(t *testing.T, w *Worker, c *testutil.Cloud, j *model.Job, files ...pikpak.File) {
	t.Helper()
	c.Files["dest"] = pikpak.File{ID: "dest", Kind: "drive#folder", Name: "作品"}
	c.Files["stage"] = pikpak.File{ID: "stage", Kind: "drive#folder", Name: j.ID}
	c.Files["root"] = pikpak.File{ID: "root", Kind: "drive#folder", ParentID: "stage", Name: "pack", Phase: "PHASE_TYPE_COMPLETE"}
	for _, f := range files {
		if f.ParentID == "" {
			f.ParentID = "root"
		}
		f.Kind = "drive#file"
		f.Phase = "PHASE_TYPE_COMPLETE"
		c.Files[f.ID] = f
	}
	j.FileID = "root"
	j.DestinationID = "dest"
	j.StagingID = "stage"
	if err := w.DB.SaveJob(context.Background(), j); err != nil {
		t.Fatal(err)
	}
}
func TestBaselineBackfillAndCrossSubscriptionDedup(t *testing.T) {
	w, c, _, f, sub, _ := setup(t)
	ctx := context.Background()
	source := "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567"
	f.items = []feed.Item{{Fingerprint: "old", Title: "作品 - 01", URL: source}}
	if err := w.Check(ctx, sub.ID, false); err != nil {
		t.Fatal(err)
	}
	jobs, _ := w.DB.Jobs(ctx, 10)
	if len(jobs) != 0 || f.resolves != 0 {
		t.Fatal("baseline downloaded existing entries")
	}
	if err := w.Check(ctx, sub.ID, true); err != nil {
		t.Fatal(err)
	}
	f.items = append(f.items, feed.Item{Fingerprint: "alias", Title: "作品 - 01", URL: source + "&dn=other-title"})
	if err := w.Check(ctx, sub.ID, false); err != nil {
		t.Fatal(err)
	}
	other := sub
	other.ID = 0
	other.Name = "另一作品"
	if err := w.SaveSubscription(ctx, &other); err != nil {
		t.Fatal(err)
	}
	if err := w.Check(ctx, other.ID, true); err != nil {
		t.Fatal(err)
	}
	jobs, _ = w.DB.Jobs(ctx, 10)
	if len(jobs) != 1 || c.Calls["submit"] != 0 {
		t.Fatal("same-account duplicate enqueued")
	}
	// Changing the feed resets its initial baseline, including colliding GUIDs.
	sub.RSSURL = "https://new.test"
	if err := w.SaveSubscription(ctx, &sub); err != nil {
		t.Fatal(err)
	}
	seen, _, _ := w.DB.Seen(ctx, sub.ID, "old")
	if seen || sub.Initialized {
		t.Fatal("new feed reused old baseline")
	}
}
func TestUncertainSubmitAndCrashNeverResubmit(t *testing.T) {
	w, c, _, _, sub, _ := setup(t)
	ctx := context.Background()
	j := enqueue(t, w, sub, "queued")
	c.Fail["submit"] = errors.New("network response lost")
	if err := w.Process(ctx, j.ID); err == nil {
		t.Fatal("expected uncertain response")
	}
	j = saved(t, w, j)
	if j.State != "submission_unknown" || c.Calls["submit"] != 1 {
		t.Fatal(j.State)
	}
	if err := w.Retry(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if c.Calls["submit"] != 1 {
		t.Fatal("uncertain submission repeated")
	}
	// The same boundary after a process crash is recovered using durable intent.
	j.State = "submitting"
	if err := w.DB.SaveJob(ctx, &j); err != nil {
		t.Fatal(err)
	}
	c.Files["finished"] = pikpak.File{ID: "finished", ParentID: j.StagingID, Name: "original.mkv", Kind: "drive#file", Phase: "PHASE_TYPE_COMPLETE"}
	if err := w.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	j = saved(t, w, j)
	if j.State != "organizing" || j.FileID != "finished" || c.Calls["submit"] != 1 {
		t.Fatal("failed to reconcile crash", j.State)
	}
}
func TestRetryRenamedButNotMovedAfterRestart(t *testing.T) {
	w, c, _, _, sub, dir := setup(t)
	ctx := context.Background()
	j := enqueue(t, w, sub, "organizing")
	prepareFiles(t, w, c, &j, pikpak.File{ID: "a", Name: "original.mkv"})
	c.Fail["move"] = errors.New("temporary disconnect")
	if err := w.Process(ctx, j.ID); err == nil {
		t.Fatal("expected move failure")
	}
	if c.Files["a"].Name != "作品 - S01E09.mkv" || c.Files["a"].ParentID == "dest" {
		t.Fatal("rename/move boundary was not exercised")
	}
	if err := w.DB.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	w.DB = db
	defer db.Close()
	if err := w.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if saved(t, w, j).State != "complete" || c.Calls["rename"] != 1 || c.Files["a"].ParentID != "dest" {
		t.Fatal("restart repeated rename or lost partial progress")
	}
}
func TestMultiFileAmbiguityAndPreservedAttachments(t *testing.T) {
	w, c, _, _, sub, _ := setup(t)
	ctx := context.Background()
	j := enqueue(t, w, sub, "organizing")
	prepareFiles(t, w, c, &j, pikpak.File{ID: "a", Name: "unparsed.mkv"}, pikpak.File{ID: "b", Name: "作品 S01E02.mkv"}, pikpak.File{ID: "c", Name: "作品 S01E02.nfo"})
	if err := w.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if saved(t, w, j).State != "needs_review" || c.Files["a"].Name != "unparsed.mkv" || c.Files["a"].ParentID != "root" {
		t.Fatal("ambiguous multi-file was renamed with RSS fallback")
	}
	if c.Files["b"].Name != "作品 - S01E02.mkv" || c.Files["b"].ParentID != "dest" || c.Files["c"].Name != "作品 S01E02.nfo" || c.Files["c"].ParentID == "root" {
		t.Fatal("per-file episode or attachment preservation failed")
	}
	// After one primary was moved, its sibling still cannot use RSS fallback.
	if err := w.Retry(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if err := w.Process(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if c.Files["a"].Name != "unparsed.mkv" {
		t.Fatal("partial retry incorrectly became a single-file download")
	}
}
func TestNameCollisionsDoNotChangeOriginals(t *testing.T) {
	for _, internalCollision := range []bool{false, true} {
		t.Run(map[bool]string{false: "existing", true: "planned"}[internalCollision], func(t *testing.T) {
			w, c, _, _, sub, _ := setup(t)
			j := enqueue(t, w, sub, "organizing")
			files := []pikpak.File{{ID: "a", Name: "作品 S01E03.mkv"}}
			if internalCollision {
				files = append(files, pikpak.File{ID: "b", Name: "Other S01E03.mkv"})
			} else {
				c.Files["existing"] = pikpak.File{ID: "existing", ParentID: "dest", Name: "作品 - S01E03.mkv"}
			}
			prepareFiles(t, w, c, &j, files...)
			if err := w.Process(context.Background(), j.ID); err != nil {
				t.Fatal(err)
			}
			if saved(t, w, j).State != "needs_review" || c.Calls["rename"] != 0 || c.Calls["move"] != 0 {
				t.Fatal("collision changed original files")
			}
		})
	}
}
func TestAuthQuotaAndRateHandling(t *testing.T) {
	for _, kind := range []string{"auth", "quota", "rate"} {
		t.Run(kind, func(t *testing.T) {
			w, c, p, _, sub, _ := setup(t)
			j := enqueue(t, w, sub, "queued")
			c.Fail["submit"] = &pikpak.Error{Kind: kind, Message: "safe test error"}
			_ = w.Process(context.Background(), j.ID)
			j = saved(t, w, j)
			if kind == "rate" {
				if j.State != "queued" || j.NextAttempt <= time.Now().Unix() || p.paused != nil {
					t.Fatal("rate limit did not back off")
				}
			} else if j.State != "paused_"+kind || p.paused == nil {
				t.Fatal("credential/quota error did not pause")
			}
		})
	}
}
func TestDifferentAccountCannotResume(t *testing.T) {
	w, c, _, _, sub, _ := setup(t)
	j := enqueue(t, w, sub, "queued")
	c.AccountID = "another-account"
	if err := w.Process(context.Background(), j.ID); err != nil {
		t.Fatal(err)
	}
	if saved(t, w, j).State != "paused_account" || c.Calls["submit"] != 0 {
		t.Fatal("old account submitted to new account")
	}
	if err := w.Retry(context.Background(), j.ID); err == nil {
		t.Fatal("allowed retry for another account")
	}
}
func TestExplicitBackfillUsesCurrentAccountDedup(t *testing.T) {
	w, c, _, f, sub, _ := setup(t)
	ctx := context.Background()
	f.items = []feed.Item{{Fingerprint: "same-guid", Title: "作品 S01E01", URL: "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567"}}
	if err := w.Check(ctx, sub.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := w.Check(ctx, sub.ID, true); err != nil {
		t.Fatal(err)
	}
	jobs, _ := w.DB.Jobs(ctx, 10)
	if len(jobs) != 1 {
		t.Fatal("same account backfill repeated a task")
	}
	c.AccountID = "new-account"
	if err := w.Check(ctx, sub.ID, true); err != nil {
		t.Fatal(err)
	}
	jobs, _ = w.DB.Jobs(ctx, 10)
	if len(jobs) != 2 {
		t.Fatal("new account's explicit backfill was suppressed by old feed fingerprints")
	}
	for _, j := range jobs {
		if j.AccountID == "new-account" && (j.TaskID != "" || j.FileID != "" || j.StagingID != "") {
			t.Fatal("new account reused another account's cloud identifiers")
		}
	}
}
