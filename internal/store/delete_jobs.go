package store

import "context"

func (s *Store) DeleteJob(ctx context.Context, id string) (int64, error) {
	return s.deleteJobs(ctx, "id=?", id)
}

func (s *Store) ClearCompletedJobs(ctx context.Context) (int64, error) {
	return s.deleteJobs(ctx, "state='complete'")
}

// Keep IDs as tombstones so retrying a previously confirmed request cannot
// resurrect a deleted download. Feed baselines and activity logs are retained.
func (s *Store) deleteJobs(ctx context.Context, selector string, args ...any) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO deleted_jobs(id) SELECT id FROM jobs WHERE "+selector, args...); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM file_actions WHERE job_id IN (SELECT id FROM jobs WHERE "+selector+")", args...); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM jobs WHERE "+selector, args...)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}
