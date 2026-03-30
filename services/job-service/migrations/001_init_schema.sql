CREATE TYPE job_status AS ENUM (
    'CREATED',
    'PENDING_METADATA',
    'PROCESSING',
    'COMPLETED',
    'FAILED',
    'STITCHING'
);

CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    s3_path TEXT NOT NULL,
    frame_start INTEGER NOT NULL,
    frame_end INTEGER NOT NULL,
    worker_count INTEGER NOT NULL,
    status job_status NOT NULL,
    result_path TEXT,
    preview_url TEXT,
    options JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    frame_start INTEGER NOT NULL,
    frame_end INTEGER NOT NULL,
    status job_status NOT NULL,
    worker_id TEXT,
    result_path TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tasks_job_id ON tasks(job_id);
