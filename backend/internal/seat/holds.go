package seat

import (
	"context"
	"errors"
	"time"
)

var ErrSeatUnavailable = errors.New("one or more seats are unavailable")
var ErrEmailUnverified = errors.New("email verification is required")

type Hold struct {
	ID          string    `json:"id"`
	ScreeningID string    `json:"screeningId"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Seats       []Seat    `json:"seats"`
}
type CheckoutHold struct {
	Hold
	MovieTitle string    `json:"movieTitle"`
	VenueName  string    `json:"venueName"`
	StartsAt   time.Time `json:"startsAt"`
	TotalPaise int       `json:"totalPaise"`
}

func (s *Store) Hold(ctx context.Context, screeningID, userID string, seatIDs []string) (Hold, error) {
	var result Hold
	if len(seatIDs) < 1 || len(seatIDs) > 10 {
		return result, errors.New("select between 1 and 10 seats")
	}
	seen := map[string]struct{}{}
	for _, id := range seatIDs {
		if id == "" {
			return result, ErrSeatUnavailable
		}
		if _, ok := seen[id]; ok {
			return result, ErrSeatUnavailable
		}
		seen[id] = struct{}{}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	var verified bool
	if err = tx.QueryRow(ctx, `SELECT email_verified_at IS NOT NULL FROM users WHERE id=$1`, userID).Scan(&verified); err != nil {
		return result, err
	}
	if !verified {
		return result, ErrEmailUnverified
	}
	var valid bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM screenings sc JOIN movies m ON m.id=sc.movie_id WHERE sc.id=$1 AND sc.status='scheduled' AND sc.starts_at>now() AND m.status='published')`, screeningID).Scan(&valid); err != nil || !valid {
		return result, ErrSeatUnavailable
	}
	_, err = tx.Exec(ctx, `UPDATE screening_seats SET status='available',held_by=NULL,hold_id=NULL,hold_expires_at=NULL WHERE screening_id=$1 AND status='held' AND hold_expires_at<=now()`, screeningID)
	if err != nil {
		return result, err
	}
	result.ScreeningID = screeningID
	result.ExpiresAt = time.Now().Add(10 * time.Minute)
	if err = tx.QueryRow(ctx, `INSERT INTO seat_holds(user_id,screening_id,expires_at) VALUES($1,$2,$3) RETURNING id::text`, userID, screeningID, result.ExpiresAt).Scan(&result.ID); err != nil {
		return result, err
	}
	rows, err := tx.Query(ctx, `UPDATE screening_seats SET status='held',held_by=$1,hold_id=$2,hold_expires_at=$3 WHERE screening_id=$4 AND id::text=ANY($5) AND status='available' RETURNING id::text,seat_key,row_label,seat_number,category,price_paise,status`, userID, result.ID, result.ExpiresAt, screeningID, seatIDs)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item Seat
		if err = rows.Scan(&item.ID, &item.Key, &item.Row, &item.Number, &item.Category, &item.PricePaise, &item.Status); err != nil {
			rows.Close()
			return result, err
		}
		result.Seats = append(result.Seats, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return result, err
	}
	if len(result.Seats) != len(seatIDs) {
		return Hold{}, ErrSeatUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return Hold{}, err
	}
	return result, nil
}

func (s *Store) Release(ctx context.Context, holdID, userID string) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var screeningID string
	err = tx.QueryRow(ctx, `UPDATE seat_holds SET status='cancelled' WHERE id=$1 AND user_id=$2 AND status='active' RETURNING screening_id::text`, holdID, userID).Scan(&screeningID)
	if err != nil {
		return "", ErrSeatUnavailable
	}
	if _, err = tx.Exec(ctx, `UPDATE waitlist_entries SET status='cancelled' WHERE id IN (SELECT waitlist_entry_id FROM screening_seats WHERE hold_id=$1 AND waitlist_entry_id IS NOT NULL) AND status='offered'`, holdID); err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `UPDATE screening_seats SET status='available',held_by=NULL,hold_id=NULL,hold_expires_at=NULL,waitlist_entry_id=NULL WHERE hold_id=$1 AND status IN ('held','waitlist_reserved')`, holdID); err != nil {
		return "", err
	}
	return screeningID, tx.Commit(ctx)
}

func (s *Store) ReleaseExpired(ctx context.Context) ([]string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT DISTINCT screening_id::text FROM seat_holds WHERE status='active' AND expires_at<=now()`)
	if err != nil {
		return nil, err
	}
	var screeningIDs []string
	for rows.Next() {
		var screeningID string
		if err := rows.Scan(&screeningID); err != nil {
			rows.Close()
			return nil, err
		}
		screeningIDs = append(screeningIDs, screeningID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE waitlist_entries SET status='expired' WHERE status='offered' AND offer_expires_at<=now()`); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE screening_seats SET status='available',held_by=NULL,hold_id=NULL,hold_expires_at=NULL,waitlist_entry_id=NULL WHERE hold_id IN (SELECT id FROM seat_holds WHERE status='active' AND expires_at<=now()) AND status IN ('held','waitlist_reserved')`); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE seat_holds SET status='expired' WHERE status='active' AND expires_at<=now()`); err != nil {
		return nil, err
	}
	return screeningIDs, tx.Commit(ctx)
}

func (s *Store) HoldDetails(ctx context.Context, holdID, userID string) (CheckoutHold, error) {
	var result CheckoutHold
	err := s.pool.QueryRow(ctx, `SELECT h.id::text,h.screening_id::text,h.expires_at,m.title,v.name,sc.starts_at FROM seat_holds h JOIN screenings sc ON sc.id=h.screening_id JOIN movies m ON m.id=sc.movie_id JOIN venues v ON v.id=sc.venue_id WHERE h.id=$1 AND h.user_id=$2 AND h.status='active' AND h.expires_at>now()`, holdID, userID).Scan(&result.ID, &result.ScreeningID, &result.ExpiresAt, &result.MovieTitle, &result.VenueName, &result.StartsAt)
	if err != nil {
		return result, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,seat_key,row_label,seat_number,category,price_paise,status FROM screening_seats WHERE hold_id=$1 AND status IN ('held','waitlist_reserved') ORDER BY row_label,seat_key`, holdID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Seat
		if err := rows.Scan(&item.ID, &item.Key, &item.Row, &item.Number, &item.Category, &item.PricePaise, &item.Status); err != nil {
			return result, err
		}
		result.Seats = append(result.Seats, item)
		result.TotalPaise += item.PricePaise
	}
	return result, rows.Err()
}
