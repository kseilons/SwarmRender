ALTER TABLE jobs ADD COLUMN IF NOT EXISTS owner_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_jobs_owner_id ON jobs(owner_id);
