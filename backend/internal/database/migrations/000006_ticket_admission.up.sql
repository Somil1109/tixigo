ALTER TABLE bookings ADD COLUMN checked_in_at TIMESTAMPTZ;
ALTER TABLE bookings ADD COLUMN checked_in_by UUID REFERENCES users(id);
CREATE INDEX bookings_admission_idx ON bookings (reference, status, checked_in_at);
