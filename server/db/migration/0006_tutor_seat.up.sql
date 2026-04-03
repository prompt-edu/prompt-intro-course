BEGIN;
ALTER TABLE seat ADD COLUMN is_tutor_seat boolean NOT NULL DEFAULT false;
COMMIT;
