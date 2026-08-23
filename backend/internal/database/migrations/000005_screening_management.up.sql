CREATE TYPE screening_status AS ENUM ('scheduled', 'cancelled');
ALTER TABLE screenings ADD COLUMN status screening_status NOT NULL DEFAULT 'scheduled';
ALTER TABLE screenings ADD COLUMN cancelled_at TIMESTAMPTZ;
CREATE INDEX screenings_organiser_status_idx ON screenings (movie_id, status, starts_at);
