package worker

import (
	"context"
	"errors"
	"reflect"

	"github.com/wade00754/pikpak-rss-manager/internal/model"
)

func SubscriptionSelection(s model.Subscription) model.Subscription {
	s.Initialized = false
	s.LastChecked = 0
	s.NextCheck = 0
	s.LastError = ""
	s.DestinationAccountRef = ""
	return s
}

// QueueSelection is called with Gate held to serialize account/destination
// validation with switching accounts; subMu also excludes subscription edits.
func (w *Worker) QueueSelection(ctx context.Context, snapshot model.Subscription, jobs []model.Job, fingerprints []string) error {
	w.subMu.Lock()
	defer w.subMu.Unlock()
	current, err := w.DB.Subscription(ctx, snapshot.ID)
	if err != nil || !reflect.DeepEqual(SubscriptionSelection(current), SubscriptionSelection(snapshot)) {
		return errors.New("訂閱已變更，請重新讀取補抓清單")
	}
	if err := w.DB.EnqueueBatch(ctx, jobs, fingerprints); err != nil {
		return err
	}
	return nil
}
