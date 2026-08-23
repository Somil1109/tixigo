package waitlist

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidEntry = errors.New("waitlist entry is invalid")
var ErrActiveEntry = errors.New("an active waitlist entry already exists")
var ErrEntryNotFound = errors.New("waitlist entry was not found")

type Entry struct {
	ID             string     `json:"id"`
	ScreeningID    string     `json:"screeningId"`
	MovieTitle     string     `json:"movieTitle"`
	VenueName      string     `json:"venueName"`
	StartsAt       time.Time  `json:"startsAt"`
	Category       string     `json:"category"`
	Quantity       int        `json:"quantity"`
	Status         string     `json:"status"`
	OfferExpiresAt *time.Time `json:"offerExpiresAt,omitempty"`
	HoldID         *string    `json:"holdId,omitempty"`
}

type Offer struct {
	EntryID       string
	ScreeningID   string
	CustomerEmail string
	MovieTitle    string
	VenueName     string
	Quantity      int
	ExpiresAt     time.Time
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Join(ctx context.Context, userID, screeningID, category string, quantity int) (Entry, error) {
	var result Entry
	if category == "" || quantity < 1 || quantity > 10 {
		return result, ErrInvalidEntry
	}
	var valid bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(
SELECT 1 FROM screenings sc
JOIN movies m ON m.id=sc.movie_id
JOIN users u ON u.id=$1
JOIN screening_seats ss ON ss.screening_id=sc.id AND ss.category=$3
WHERE sc.id=$2 AND sc.starts_at>now() AND m.status='published' AND u.email_verified_at IS NOT NULL
)`, userID, screeningID, category).Scan(&valid)
	if err != nil || !valid {
		return result, ErrInvalidEntry
	}
	err = s.pool.QueryRow(ctx, `INSERT INTO waitlist_entries(user_id,screening_id,category,quantity) VALUES($1,$2,$3,$4) RETURNING id::text,screening_id::text,category,quantity,status::text`, userID, screeningID, category, quantity).Scan(&result.ID, &result.ScreeningID, &result.Category, &result.Quantity, &result.Status)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return result, ErrActiveEntry
		}
		return result, err
	}
	return result, nil
}

func (s *Store) List(ctx context.Context, userID string) ([]Entry, error) {
	rows, err := s.pool.Query(ctx, `SELECT w.id::text,w.screening_id::text,m.title,v.name,sc.starts_at,w.category,w.quantity,w.status::text,w.offer_expires_at,
(SELECT sh.id::text FROM seat_holds sh JOIN screening_seats ss ON ss.hold_id=sh.id WHERE ss.waitlist_entry_id=w.id LIMIT 1)
FROM waitlist_entries w JOIN screenings sc ON sc.id=w.screening_id JOIN movies m ON m.id=sc.movie_id JOIN venues v ON v.id=sc.venue_id
WHERE w.user_id=$1 ORDER BY w.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Entry{}
	for rows.Next() {
		var item Entry
		if err := rows.Scan(&item.ID, &item.ScreeningID, &item.MovieTitle, &item.VenueName, &item.StartsAt, &item.Category, &item.Quantity, &item.Status, &item.OfferExpiresAt, &item.HoldID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Cancel(ctx context.Context, entryID, userID string) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var screeningID string
	err = tx.QueryRow(ctx, `UPDATE waitlist_entries SET status='cancelled' WHERE id=$1 AND user_id=$2 AND status IN ('waiting','offered') RETURNING screening_id::text`, entryID, userID).Scan(&screeningID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrEntryNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `UPDATE seat_holds SET status='cancelled' WHERE id IN (SELECT hold_id FROM screening_seats WHERE waitlist_entry_id=$1) AND status='active'`, entryID); err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `UPDATE screening_seats SET status='available',held_by=NULL,hold_id=NULL,hold_expires_at=NULL,waitlist_entry_id=NULL WHERE waitlist_entry_id=$1 AND status='waitlist_reserved'`, entryID); err != nil {
		return "", err
	}
	return screeningID, tx.Commit(ctx)
}

func (s *Store) Match(ctx context.Context, screeningID string) ([]Offer, error) {
	offers := []Offer{}
	for {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return offers, err
		}
		var offer Offer
		var category, userID string
		err = tx.QueryRow(ctx, `SELECT w.id::text,w.user_id::text,w.screening_id::text,w.category,w.quantity,u.email,m.title,v.name
FROM waitlist_entries w JOIN users u ON u.id=w.user_id JOIN screenings sc ON sc.id=w.screening_id JOIN movies m ON m.id=sc.movie_id JOIN venues v ON v.id=sc.venue_id
WHERE w.screening_id=$1 AND w.status='waiting' ORDER BY w.created_at LIMIT 1 FOR UPDATE OF w SKIP LOCKED`, screeningID).Scan(&offer.EntryID, &userID, &offer.ScreeningID, &category, &offer.Quantity, &offer.CustomerEmail, &offer.MovieTitle, &offer.VenueName)
		if errors.Is(err, pgx.ErrNoRows) {
			tx.Rollback(ctx)
			return offers, nil
		}
		if err != nil {
			tx.Rollback(ctx)
			return offers, err
		}
		rows, err := tx.Query(ctx, `SELECT id::text FROM screening_seats WHERE screening_id=$1 AND category=$2 AND status='available' ORDER BY row_label,seat_key LIMIT $3 FOR UPDATE SKIP LOCKED`, screeningID, category, offer.Quantity)
		if err != nil {
			tx.Rollback(ctx)
			return offers, err
		}
		seatIDs := []string{}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				tx.Rollback(ctx)
				return offers, err
			}
			seatIDs = append(seatIDs, id)
		}
		rows.Close()
		if len(seatIDs) < offer.Quantity {
			tx.Rollback(ctx)
			return offers, nil
		}
		offer.ExpiresAt = time.Now().Add(time.Hour)
		var holdID string
		if err = tx.QueryRow(ctx, `INSERT INTO seat_holds(user_id,screening_id,expires_at) VALUES($1,$2,$3) RETURNING id::text`, userID, screeningID, offer.ExpiresAt).Scan(&holdID); err != nil {
			tx.Rollback(ctx)
			return offers, err
		}
		if _, err = tx.Exec(ctx, `UPDATE screening_seats SET status='waitlist_reserved',held_by=$1,hold_id=$2,hold_expires_at=$3,waitlist_entry_id=$4 WHERE id::text=ANY($5)`, userID, holdID, offer.ExpiresAt, offer.EntryID, seatIDs); err != nil {
			tx.Rollback(ctx)
			return offers, err
		}
		if _, err = tx.Exec(ctx, `UPDATE waitlist_entries SET status='offered',offered_at=now(),offer_expires_at=$2 WHERE id=$1`, offer.EntryID, offer.ExpiresAt); err != nil {
			tx.Rollback(ctx)
			return offers, err
		}
		if err = tx.Commit(ctx); err != nil {
			return offers, err
		}
		offers = append(offers, offer)
	}
}
