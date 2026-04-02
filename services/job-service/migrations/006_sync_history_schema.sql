
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS completed_frames INTEGER DEFAULT 0;

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS current_frame INTEGER DEFAULT 0;

DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tasks_history' AND column_name='type') THEN
        ALTER TABLE tasks_history ADD COLUMN type task_type NOT NULL DEFAULT 'RENDER';
    END IF;
END $$;

ALTER TABLE tasks_history ADD COLUMN IF NOT EXISTS current_frame INTEGER DEFAULT 0;

ALTER TABLE tasks_history ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tasks_history ADD COLUMN IF NOT EXISTS sys_period tstzrange NOT NULL DEFAULT tstzrange(current_timestamp, null);
