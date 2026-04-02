-- Add logs column to tasks and tasks_history tables
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS logs TEXT;
ALTER TABLE tasks_history ADD COLUMN IF NOT EXISTS logs TEXT;
