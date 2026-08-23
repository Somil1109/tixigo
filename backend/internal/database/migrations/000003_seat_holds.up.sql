CREATE TYPE hold_status AS ENUM ('active', 'completed', 'expired', 'cancelled');

CREATE TABLE seat_holds (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  screening_id UUID NOT NULL REFERENCES screenings(id) ON DELETE CASCADE,
  status hold_status NOT NULL DEFAULT 'active',
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE screening_seats ADD COLUMN hold_id UUID REFERENCES seat_holds(id) ON DELETE SET NULL;
CREATE INDEX seat_holds_expiry_idx ON seat_holds (expires_at) WHERE status='active';
CREATE INDEX screening_seats_hold_idx ON screening_seats (hold_id) WHERE hold_id IS NOT NULL;
