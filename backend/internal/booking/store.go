package booking

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tixigo/tixigo-api/internal/seat"
	"time"
)

var ErrHoldUnavailable = errors.New("hold is unavailable or expired")

type Booking struct {
	ID            string      `json:"id"`
	Reference     string      `json:"reference"`
	Status        string      `json:"status"`
	MovieTitle    string      `json:"movieTitle"`
	VenueName     string      `json:"venueName"`
	StartsAt      time.Time   `json:"startsAt"`
	TotalPaise    int         `json:"totalPaise"`
	Seats         []seat.Seat `json:"seats"`
	CustomerEmail string      `json:"-"`
}
type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool} }
func reference() (string, error) {
	value := make([]byte, 7)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "TIX-" + base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(value), nil
}
func (s *Store) Confirm(ctx context.Context, holdID, userID string) (Booking, error) {
	var result Booking
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	var screeningID string
	var expires time.Time
	var holdStatus string
	err = tx.QueryRow(ctx, `SELECT h.screening_id::text,h.expires_at,h.status,u.email,m.title,v.name,sc.starts_at FROM seat_holds h JOIN users u ON u.id=h.user_id JOIN screenings sc ON sc.id=h.screening_id JOIN movies m ON m.id=sc.movie_id JOIN venues v ON v.id=sc.venue_id WHERE h.id=$1 AND h.user_id=$2 FOR UPDATE OF h`, holdID, userID).Scan(&screeningID, &expires, &holdStatus, &result.CustomerEmail, &result.MovieTitle, &result.VenueName, &result.StartsAt)
	if err != nil || holdStatus != "active" || !expires.After(time.Now()) {
		return result, ErrHoldUnavailable
	}
	rows, err := tx.Query(ctx, `SELECT id::text,seat_key,row_label,seat_number,category,price_paise,status FROM screening_seats WHERE hold_id=$1 AND status='held' FOR UPDATE`, holdID)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item seat.Seat
		if err := rows.Scan(&item.ID, &item.Key, &item.Row, &item.Number, &item.Category, &item.PricePaise, &item.Status); err != nil {
			rows.Close()
			return result, err
		}
		result.Seats = append(result.Seats, item)
		result.TotalPaise += item.PricePaise
	}
	rows.Close()
	if len(result.Seats) == 0 {
		return result, ErrHoldUnavailable
	}
	result.Reference, err = reference()
	if err != nil {
		return result, err
	}
	err = tx.QueryRow(ctx, `INSERT INTO bookings(reference,user_id,screening_id,total_paise) VALUES($1,$2,$3,$4) RETURNING id::text,status`, result.Reference, userID, screeningID, result.TotalPaise).Scan(&result.ID, &result.Status)
	if err != nil {
		return result, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO booking_seats(booking_id,screening_seat_id) SELECT $1,id FROM screening_seats WHERE hold_id=$2`, result.ID, holdID); err != nil {
		return result, err
	}
	if _, err = tx.Exec(ctx, `UPDATE screening_seats SET status='booked',held_by=NULL,hold_expires_at=NULL,hold_id=NULL WHERE hold_id=$1 AND status='held'`, holdID); err != nil {
		return result, err
	}
	if _, err = tx.Exec(ctx, `UPDATE seat_holds SET status='completed' WHERE id=$1`, holdID); err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return result, err
	}
	return result, nil
}
