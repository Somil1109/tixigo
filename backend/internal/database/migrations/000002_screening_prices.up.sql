CREATE TABLE screening_category_prices (
  screening_id UUID NOT NULL REFERENCES screenings(id) ON DELETE CASCADE,
  category TEXT NOT NULL,
  price_paise INTEGER NOT NULL CHECK (price_paise > 0),
  PRIMARY KEY (screening_id, category)
);
