package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed migrations/001_init.sql
var migration string

//go:embed migrations/002_web_setup.sql
var webSetupMigration string

//go:embed migrations/003_repeat_downloads.sql
var repeatDownloadsMigration string

//go:embed migrations/004_deleted_jobs.sql
var deletedJobsMigration string

type Store struct {
	db   *sql.DB
	aead cipher.AEAD
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	keyPath := filepath.Join(dir, "secret.key")
	key, err := os.ReadFile(keyPath)
	if errors.Is(err, os.ErrNotExist) {
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return nil, err
		}
		f, e := os.OpenFile(keyPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return nil, e
		}
		_, err = f.Write(key)
		closeErr := f.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
	} else if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("資料金鑰長度不正確；請保留原金鑰")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dir, "manager.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	var schemaVersion int
	if err := db.QueryRow("PRAGMA user_version").Scan(&schemaVersion); err != nil {
		db.Close()
		return nil, err
	}
	if schemaVersion > 4 {
		db.Close()
		return nil, errors.New("資料庫由較新版本建立，請使用對應版本服務")
	}
	s := &Store{db, aead}
	statements := []string{"PRAGMA journal_mode=WAL", "PRAGMA foreign_keys=ON", "PRAGMA busy_timeout=5000"}
	if schemaVersion < 1 {
		statements = append(statements, migration)
	}
	if schemaVersion < 2 {
		statements = append(statements, webSetupMigration)
	}
	if schemaVersion < 3 {
		statements = append(statements, "PRAGMA foreign_keys=OFF", repeatDownloadsMigration, "PRAGMA foreign_keys=ON")
	}
	if schemaVersion < 4 {
		statements = append(statements, deletedJobsMigration)
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			return nil, err
		}
	}
	_ = os.Chmod(dbPath, 0600)
	return s, nil
}

func (s *Store) Close() error                   { return s.db.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func ID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func (s *Store) seal(v string) string {
	n := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(n); err != nil {
		panic(err)
	}
	return "enc:" + base64.RawStdEncoding.EncodeToString(s.aead.Seal(n, n, []byte(v), nil))
}
func (s *Store) open(v string) (string, error) {
	if !strings.HasPrefix(v, "enc:") {
		return "", errors.New("未加密的私密設定")
	}
	b, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(v, "enc:"))
	if err != nil || len(b) < s.aead.NonceSize() {
		return "", errors.New("私密設定損毀")
	}
	p, err := s.aead.Open(nil, b[:s.aead.NonceSize()], b[s.aead.NonceSize():], nil)
	if err != nil {
		return "", errors.New("私密設定無法解密；請保留原金鑰")
	}
	return string(p), nil
}

func (s *Store) Setting(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key=?", key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return s.open(v)
}
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, s.seal(value))
	return err
}

func (s *Store) subscriptionPayload(sub model.Subscription) ([]byte, error) {
	sub.RSSURL = s.seal(sub.RSSURL)
	sub.DestinationAccountRef = ""
	return json.Marshal(model.SubscriptionRecord{Subscription: sub, StoredDestinationAccountID: sub.DestinationAccountID})
}
func (s *Store) decodeSubscription(payload string) (model.Subscription, error) {
	var record model.SubscriptionRecord
	if err := json.Unmarshal([]byte(payload), &record); err != nil {
		return record.Subscription, err
	}
	sub := record.Subscription
	sub.DestinationAccountID = record.StoredDestinationAccountID
	u, err := s.open(sub.RSSURL)
	sub.RSSURL = u
	return sub, err
}
func (s *Store) SaveSubscription(ctx context.Context, sub *model.Subscription) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	p, err := s.subscriptionPayload(*sub)
	if err != nil {
		return err
	}
	if sub.ID == 0 {
		result, err := tx.ExecContext(ctx, "INSERT INTO subscriptions(enabled,next_check,payload) VALUES(?,?,?)", sub.Enabled, sub.NextCheck, string(p))
		if err != nil {
			return err
		}
		sub.ID, err = result.LastInsertId()
		if err != nil {
			return err
		}
		p, err = s.subscriptionPayload(*sub)
		if err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE subscriptions SET enabled=?,next_check=?,payload=? WHERE id=?", sub.Enabled, sub.NextCheck, string(p), sub.ID); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) ResetFeed(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM seen_items WHERE subscription_id=?", id)
	return err
}
func (s *Store) Subscription(ctx context.Context, id int64) (model.Subscription, error) {
	var p string
	if err := s.db.QueryRowContext(ctx, "SELECT payload FROM subscriptions WHERE id=?", id).Scan(&p); err != nil {
		return model.Subscription{}, err
	}
	return s.decodeSubscription(p)
}
func (s *Store) Subscriptions(ctx context.Context) ([]model.Subscription, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT payload FROM subscriptions ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Subscription{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		sub, err := s.decodeSubscription(p)
		if err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}
func (s *Store) DeleteSubscription(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM subscriptions WHERE id=?", id)
	return err
}
func (s *Store) Seen(ctx context.Context, id int64, fingerprint string) (bool, bool, error) {
	var b bool
	err := s.db.QueryRowContext(ctx, "SELECT baseline FROM seen_items WHERE subscription_id=? AND fingerprint=?", id, fingerprint).Scan(&b)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	return err == nil, b, err
}
func (s *Store) Baseline(ctx context.Context, sub *model.Subscription, fingerprints []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, fingerprint := range fingerprints {
		if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO seen_items(subscription_id,fingerprint,baseline) VALUES(?,?,1)", sub.ID, fingerprint); err != nil {
			return err
		}
	}
	sub.Initialized = true
	p, err := s.subscriptionPayload(*sub)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE subscriptions SET payload=?,enabled=?,next_check=? WHERE id=?", string(p), sub.Enabled, sub.NextCheck, sub.ID); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) jobPayload(j model.Job) ([]byte, error) {
	return json.Marshal(model.JobRecord{Job: j, StoredAccountID: j.AccountID, StoredResourceURL: s.seal(j.ResourceURL)})
}
func (s *Store) decodeJob(p string) (model.Job, error) {
	var r model.JobRecord
	if err := json.Unmarshal([]byte(p), &r); err != nil {
		return r.Job, err
	}
	r.Job.AccountID = r.StoredAccountID
	u, err := s.open(r.StoredResourceURL)
	r.Job.ResourceURL = u
	return r.Job, err
}
func (s *Store) Enqueue(ctx context.Context, j model.Job, fingerprint string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	added, err := s.enqueueTx(ctx, tx, j, fingerprint)
	if err != nil {
		return false, err
	}
	return added, tx.Commit()
}

// EnqueueBatch commits a confirmed selection atomically. Stable job IDs make
// retries of the same confirmation idempotent without suppressing redownloads.
func (s *Store) EnqueueBatch(ctx context.Context, jobs []model.Job, fingerprints []string) error {
	if len(jobs) != len(fingerprints) {
		return errors.New("無法建立任務")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, j := range jobs {
		if _, err := s.enqueueTx(ctx, tx, j, fingerprints[i]); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) enqueueTx(ctx context.Context, tx *sql.Tx, j model.Job, fingerprint string) (bool, error) {
	var deleted bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM deleted_jobs WHERE id=?)", j.ID).Scan(&deleted); err != nil {
		return false, err
	}
	if deleted {
		return false, nil
	}
	p, err := s.jobPayload(j)
	if err != nil {
		return false, err
	}
	r, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO jobs(id,account_id,resource_key,subscription_id,state,next_attempt,created_at,payload) VALUES(?,?,?,?,?,?,?,?)", j.ID, j.AccountID, j.ResourceKey, j.SubscriptionID, j.State, j.NextAttempt, j.CreatedAt, string(p))
	if err != nil {
		return false, err
	}
	// Manual jobs have no subscription and must not change feed baselines.
	if j.SubscriptionID != 0 {
		if _, err := tx.ExecContext(ctx, "INSERT INTO seen_items(subscription_id,fingerprint,baseline) VALUES(?,?,0) ON CONFLICT(subscription_id,fingerprint) DO UPDATE SET baseline=0", j.SubscriptionID, fingerprint); err != nil {
			return false, err
		}
	}
	n, err := r.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
func (s *Store) SaveJob(ctx context.Context, j *model.Job) error {
	j.UpdatedAt = time.Now().Unix()
	p, err := s.jobPayload(*j)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE jobs SET state=?,next_attempt=?,payload=? WHERE id=?", j.State, j.NextAttempt, string(p), j.ID)
	return err
}
func (s *Store) Job(ctx context.Context, id string) (model.Job, error) {
	var p string
	if err := s.db.QueryRowContext(ctx, "SELECT payload FROM jobs WHERE id=?", id).Scan(&p); err != nil {
		return model.Job{}, err
	}
	return s.decodeJob(p)
}
func (s *Store) Jobs(ctx context.Context, limit int) ([]model.Job, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT payload FROM jobs ORDER BY created_at DESC,id DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Job{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		j, err := s.decodeJob(p)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}
func (s *Store) DueJobs(ctx context.Context, account string, now int64) ([]model.Job, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT payload FROM jobs WHERE account_id=? AND state IN ('queued','submitting','downloading','organizing') AND next_attempt<=? ORDER BY created_at LIMIT 10", account, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Job{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		j, err := s.decodeJob(p)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}
func (s *Store) Actions(ctx context.Context, job string) ([]model.FileAction, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT payload FROM file_actions WHERE job_id=? ORDER BY file_id", job)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.FileAction{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		var a model.FileAction
		if err := json.Unmarshal([]byte(p), &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (s *Store) SaveAction(ctx context.Context, a model.FileAction) error {
	p, err := json.Marshal(a)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO file_actions(job_id,file_id,payload) VALUES(?,?,?) ON CONFLICT(job_id,file_id) DO UPDATE SET payload=excluded.payload", a.JobID, a.FileID, string(p))
	return err
}
func (s *Store) Event(ctx context.Context, job string, sub int64, level, message string) error {
	if len(message) > 1000 {
		message = message[:1000]
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO events(job_id,subscription_id,level,message,created_at) VALUES(?,?,?,?,?)", job, sub, level, message, time.Now().Unix())
	return err
}
func (s *Store) Events(ctx context.Context) ([]model.Event, error) {
	_, _ = s.db.ExecContext(ctx, "DELETE FROM events WHERE created_at<?", time.Now().AddDate(0, 0, -30).Unix())
	rows, err := s.db.QueryContext(ctx, "SELECT id,job_id,subscription_id,level,message,created_at FROM events ORDER BY id DESC LIMIT 150")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Event{}
	for rows.Next() {
		var e model.Event
		if err := rows.Scan(&e.ID, &e.JobID, &e.SubscriptionID, &e.Level, &e.Message, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
