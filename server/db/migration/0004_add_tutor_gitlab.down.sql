BEGIN;

DROP TABLE IF EXISTS student_gitlab_processes;

ALTER TABLE tutor DROP COLUMN IF EXISTS gitlab_username;

COMMIT;
