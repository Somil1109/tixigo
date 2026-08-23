DROP INDEX IF EXISTS screening_seats_waitlist_entry_idx;
DROP INDEX IF EXISTS waitlist_entries_active_customer_idx;

ALTER TABLE waitlist_entries
  ADD CONSTRAINT waitlist_entries_user_id_screening_id_category_status_key
  UNIQUE (user_id, screening_id, category, status);
