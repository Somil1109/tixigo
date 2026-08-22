CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE user_role AS ENUM ('customer', 'organiser', 'admin');
CREATE TYPE event_status AS ENUM ('draft', 'pending_approval', 'published', 'rejected');
CREATE TYPE seat_status AS ENUM ('available', 'held', 'booked', 'waitlist_reserved');
CREATE TYPE booking_status AS ENUM ('confirmed', 'cancelled');
CREATE TYPE waitlist_status AS ENUM ('waiting', 'offered', 'fulfilled', 'expired', 'cancelled');
CREATE TYPE account_token_purpose AS ENUM ('verify_email', 'reset_password');

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  full_name TEXT NOT NULL,
  role user_role NOT NULL DEFAULT 'customer',
  email_verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE account_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  purpose account_token_purpose NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE venues (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  address TEXT NOT NULL,
  city TEXT NOT NULL,
  timezone TEXT NOT NULL DEFAULT 'Asia/Kolkata',
  layout JSONB NOT NULL,
  created_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE movies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  poster_url TEXT NOT NULL,
  trailer_url TEXT,
  genre TEXT[] NOT NULL DEFAULT '{}',
  language TEXT NOT NULL,
  duration_minutes INTEGER NOT NULL CHECK (duration_minutes > 0),
  age_rating TEXT NOT NULL,
  status event_status NOT NULL DEFAULT 'draft',
  rejection_reason TEXT,
  organiser_id UUID NOT NULL REFERENCES users(id),
  approved_by UUID REFERENCES users(id),
  approved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE screenings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  movie_id UUID NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
  venue_id UUID NOT NULL REFERENCES venues(id),
  starts_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (venue_id, starts_at)
);

CREATE TABLE screening_seats (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  screening_id UUID NOT NULL REFERENCES screenings(id) ON DELETE CASCADE,
  seat_key TEXT NOT NULL,
  row_label TEXT NOT NULL,
  seat_number TEXT NOT NULL,
  category TEXT NOT NULL,
  price_paise INTEGER NOT NULL CHECK (price_paise >= 0),
  status seat_status NOT NULL DEFAULT 'available',
  hold_expires_at TIMESTAMPTZ,
  held_by UUID REFERENCES users(id),
  waitlist_entry_id UUID,
  UNIQUE (screening_id, seat_key)
);

CREATE TABLE bookings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reference TEXT NOT NULL UNIQUE,
  user_id UUID NOT NULL REFERENCES users(id),
  screening_id UUID NOT NULL REFERENCES screenings(id),
  status booking_status NOT NULL DEFAULT 'confirmed',
  total_paise INTEGER NOT NULL CHECK (total_paise >= 0),
  cancelled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE booking_seats (
  booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  screening_seat_id UUID NOT NULL REFERENCES screening_seats(id),
  PRIMARY KEY (booking_id, screening_seat_id)
);

CREATE TABLE waitlist_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  screening_id UUID NOT NULL REFERENCES screenings(id) ON DELETE CASCADE,
  category TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity BETWEEN 1 AND 10),
  status waitlist_status NOT NULL DEFAULT 'waiting',
  offered_at TIMESTAMPTZ,
  offer_expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, screening_id, category, status)
);

CREATE INDEX screening_seats_status_idx ON screening_seats (screening_id, status, category);
CREATE INDEX waitlist_matching_idx ON waitlist_entries (screening_id, category, status, created_at);
CREATE INDEX screenings_starts_at_idx ON screenings (starts_at);
CREATE INDEX refresh_tokens_active_idx ON refresh_tokens (user_id, expires_at) WHERE revoked_at IS NULL;
CREATE INDEX account_tokens_active_idx ON account_tokens (user_id, purpose, expires_at) WHERE consumed_at IS NULL;
