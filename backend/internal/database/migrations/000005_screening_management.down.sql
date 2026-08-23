DROP INDEX IF EXISTS screenings_organiser_status_idx;
ALTER TABLE screenings DROP COLUMN IF EXISTS cancelled_at;
ALTER TABLE screenings DROP COLUMN IF EXISTS status;
DROP TYPE IF EXISTS screening_status;
