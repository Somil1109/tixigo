DROP INDEX IF EXISTS bookings_admission_idx;
ALTER TABLE bookings DROP COLUMN IF EXISTS checked_in_by;
ALTER TABLE bookings DROP COLUMN IF EXISTS checked_in_at;
