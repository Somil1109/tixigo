ALTER TABLE waitlist_entries
  DROP CONSTRAINT IF EXISTS waitlist_entries_user_id_screening_id_category_status_key;

CREATE UNIQUE INDEX waitlist_entries_active_customer_idx
  ON waitlist_entries (user_id, screening_id, category)
  WHERE status IN ('waiting', 'offered');

CREATE INDEX screening_seats_waitlist_entry_idx
  ON screening_seats (waitlist_entry_id)
  WHERE waitlist_entry_id IS NOT NULL;
