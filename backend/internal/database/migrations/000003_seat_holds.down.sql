ALTER TABLE screening_seats DROP COLUMN IF EXISTS hold_id;
DROP TABLE IF EXISTS seat_holds;
DROP TYPE IF EXISTS hold_status;
