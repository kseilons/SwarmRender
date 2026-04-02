CREATE TYPE job_status AS ENUM (
    'CREATED',
    'PENDING_METADATA',
    'PROCESSING',
    'COMPLETED',
    'FAILED',
    'STITCHING'
);

CREATE TYPE task_type AS ENUM (
    'RENDER',
    'STITCHING',
    'METADATA'
);

CREATE OR REPLACE FUNCTION versioning()
RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    NEW.sys_period = tstzrange(current_timestamp, null);
    RETURN NEW;
  ELSIF TG_OP = 'UPDATE' THEN
    IF OLD.sys_period = NEW.sys_period THEN
       NEW.sys_period = tstzrange(current_timestamp, null);
    END IF;
    EXECUTE format('INSERT INTO %I_history SELECT ($1).*', TG_TABLE_NAME) USING OLD;
    RETURN NEW;
  ELSIF TG_OP = 'DELETE' THEN
    EXECUTE format('INSERT INTO %I_history SELECT ($1).*', TG_TABLE_NAME) USING OLD;
    RETURN OLD;
  END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    main_file_url TEXT NOT NULL,
    frame_start INTEGER NOT NULL,
    frame_end INTEGER NOT NULL,
    worker_count INTEGER NOT NULL,
    status job_status NOT NULL,
    result_url TEXT,
    preview_url TEXT,
    options JSONB NOT NULL DEFAULT '{}',
    completed_frames INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    type task_type NOT NULL DEFAULT 'RENDER',
    frame_start INTEGER NOT NULL,
    frame_end INTEGER NOT NULL,
    status job_status NOT NULL,
    worker_id TEXT,
    result_url TEXT,
    current_frame INTEGER DEFAULT 0,
    retry_count INTEGER NOT NULL DEFAULT 0,
    sys_period tstzrange NOT NULL DEFAULT tstzrange(current_timestamp, null),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tasks_history (LIKE tasks);

CREATE INDEX IF NOT EXISTS idx_tasks_job_id ON tasks(job_id);

DROP TRIGGER IF EXISTS tasks_versioning_trigger ON tasks;
CREATE TRIGGER tasks_versioning_trigger
BEFORE UPDATE OR DELETE ON tasks
FOR EACH ROW EXECUTE PROCEDURE versioning();
