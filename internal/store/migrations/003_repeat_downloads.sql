BEGIN IMMEDIATE;
CREATE TABLE jobs_v3 (id TEXT PRIMARY KEY, account_id TEXT NOT NULL, resource_key TEXT NOT NULL, subscription_id INTEGER NOT NULL, state TEXT NOT NULL, next_attempt INTEGER NOT NULL, created_at INTEGER NOT NULL, payload TEXT NOT NULL);
INSERT INTO jobs_v3 SELECT * FROM jobs;
DROP TABLE jobs;
ALTER TABLE jobs_v3 RENAME TO jobs;
CREATE INDEX jobs_due ON jobs(state,next_attempt);
PRAGMA user_version=3;
COMMIT;
