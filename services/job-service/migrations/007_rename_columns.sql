
ALTER TABLE jobs RENAME COLUMN s3_path TO main_file_url;
ALTER TABLE jobs RENAME COLUMN result_path TO result_url;

ALTER TABLE tasks RENAME COLUMN result_path TO result_url;
ALTER TABLE tasks_history RENAME COLUMN result_path TO result_url;

UPDATE jobs SET result_url = REPLACE(result_url, 's3://blender-renders/', '') WHERE result_url LIKE 's3://%';
UPDATE jobs SET preview_url = REPLACE(preview_url, 's3://blender-renders/', '') WHERE preview_url LIKE 's3://%';
UPDATE tasks SET result_url = REPLACE(result_url, 's3://blender-renders/', '') WHERE result_url LIKE 's3://%';
UPDATE tasks_history SET result_url = REPLACE(result_url, 's3://blender-renders/', '') WHERE result_url LIKE 's3://%';
