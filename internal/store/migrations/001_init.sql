CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS subscriptions (id INTEGER PRIMARY KEY AUTOINCREMENT, enabled INTEGER NOT NULL, next_check INTEGER NOT NULL, payload TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS seen_items (subscription_id INTEGER NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE, fingerprint TEXT NOT NULL, baseline INTEGER NOT NULL DEFAULT 0, PRIMARY KEY(subscription_id,fingerprint));
CREATE TABLE IF NOT EXISTS jobs (id TEXT PRIMARY KEY, account_id TEXT NOT NULL, resource_key TEXT NOT NULL, subscription_id INTEGER NOT NULL, state TEXT NOT NULL, next_attempt INTEGER NOT NULL, created_at INTEGER NOT NULL, payload TEXT NOT NULL, UNIQUE(account_id,resource_key));
CREATE INDEX IF NOT EXISTS jobs_due ON jobs(state,next_attempt);
CREATE TABLE IF NOT EXISTS file_actions (job_id TEXT NOT NULL REFERENCES jobs(id), file_id TEXT NOT NULL, payload TEXT NOT NULL, PRIMARY KEY(job_id,file_id));
CREATE TABLE IF NOT EXISTS events (id INTEGER PRIMARY KEY AUTOINCREMENT, job_id TEXT NOT NULL DEFAULT '', subscription_id INTEGER NOT NULL DEFAULT 0, level TEXT NOT NULL, message TEXT NOT NULL, created_at INTEGER NOT NULL);
PRAGMA user_version=1;
