CREATE OR REPLACE FUNCTION versioning()
RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    NEW.sys_period = tstzrange(current_timestamp, null);
    RETURN NEW;
  ELSIF TG_OP = 'UPDATE' THEN
    -- If sys_period hasn't been updated manually, update it to [now, null)
    -- This handles the "closing" of the previous interval implicitly
    IF OLD.sys_period = NEW.sys_period THEN
       NEW.sys_period = tstzrange(current_timestamp, null);
    END IF;
    
    -- Insert old row into history table
    -- The history table contains the [valid_from, valid_to) interval
    -- where valid_to is the current_timestamp (when it was updated)
    EXECUTE format('INSERT INTO %I_history SELECT ($1).*', TG_TABLE_NAME) USING OLD;
    RETURN NEW;
  ELSIF TG_OP = 'DELETE' THEN
    -- Insert old row into history table before deletion
    EXECUTE format('INSERT INTO %I_history SELECT ($1).*', TG_TABLE_NAME) USING OLD;
    RETURN OLD;
  END IF;
END;
$$ LANGUAGE plpgsql;

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS sys_period tstzrange NOT NULL DEFAULT tstzrange(current_timestamp, null);
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS tasks_history (LIKE tasks);

DROP TRIGGER IF EXISTS tasks_versioning_trigger ON tasks;
CREATE TRIGGER tasks_versioning_trigger
BEFORE UPDATE OR DELETE ON tasks
FOR EACH ROW EXECUTE PROCEDURE versioning();
