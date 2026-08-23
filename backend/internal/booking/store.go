package booking

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tixigo/tixigo-api/internal/seat"
)

var ErrHoldUnavailable = errors.New("hold is unavailable or expired")
var ErrBookingNotFound = errors.New("booking was not found")
var ErrCancellationClosed = errors.New("booking can only be cancelled more than 24 hours before the screening")

type Booking struct {
	ID            string      `json:"id"`
	Reference     string      `json:"reference"`
	ScreeningID   string      `json:"screeningId"`
	Status        string      `json:"status"`
	MovieTitle    string      `json:"movieTitle"`
	VenueName     string      `json:"venueName"`
	StartsAt      time.Time   `json:"startsAt"`
	TotalPaise    int         `json:"totalPaise"`
	Seats         []seat.Seat `json:"seats"`
	CancelledAt   *time.Time  `json:"cancelledAt,omitempty"`
	CanCancel     bool        `json:"canCancel"`
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
	var expires time.Time
	var holdStatus string
	err = tx.QueryRow(ctx, `SELECT h.screening_id::text,h.expires_at,h.status,u.email,m.title,v.name,sc.starts_at FROM seat_holds h JOIN users u ON u.id=h.user_id JOIN screenings sc ON sc.id=h.screening_id JOIN movies m ON m.id=sc.movie_id JOIN venues v ON v.id=sc.venue_id WHERE h.id=$1 AND h.user_id=$2 FOR UPDATE OF h`, holdID, userID).Scan(&result.ScreeningID, &expires, &holdStatus, &result.CustomerEmail, &result.MovieTitle, &result.VenueName, &result.StartsAt)
	if err != nil || holdStatus != "active" || !expires.After(time.Now()) {
		return result, ErrHoldUnavailable
	}
	rows, err := tx.Query(ctx, `SELECT id::text,seat_key,row_label,seat_number,category,price_paise,status FROM screening_seats WHERE hold_id=$1 AND status IN ('held','waitlist_reserved') FOR UPDATE`, holdID)
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
	err = tx.QueryRow(ctx, `INSERT INTO bookings(reference,user_id,screening_id,total_paise) VALUES($1,$2,$3,$4) RETURNING id::text,status`, result.Reference, userID, result.ScreeningID, result.TotalPaise).Scan(&result.ID, &result.Status)
	if err != nil {
		return result, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO booking_seats(booking_id,screening_seat_id) SELECT $1,id FROM screening_seats WHERE hold_id=$2`, result.ID, holdID); err != nil {
		return result, err
	}
	if _, err = tx.Exec(ctx, `UPDATE waitlist_entries SET status='fulfilled' WHERE id IN (SELECT waitlist_entry_id FROM screening_seats WHERE hold_id=$1 AND waitlist_entry_id IS NOT NULL)`, holdID); err != nil {
		return result, err
	}
	if _, err = tx.Exec(ctx, `UPDATE screening_seats SET status='booked',held_by=NULL,hold_expires_at=NULL,hold_id=NULL,waitlist_entry_id=NULL WHERE hold_id=$1 AND status IN ('held','waitlist_reserved')`, holdID); err != nil {
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

const bookingSelect = `SELECT b.id::text,b.reference,b.screening_id::text,b.status::text,m.title,v.name,sc.starts_at,b.total_paise,b.cancelled_at,u.email,
ss.id::text,ss.seat_key,ss.row_label,ss.seat_number,ss.category,ss.price_paise,ss.status::text
FROM bookings b
JOIN users u ON u.id=b.user_id
JOIN screenings sc ON sc.id=b.screening_id
JOIN movies m ON m.id=sc.movie_id
JOIN venues v ON v.id=sc.venue_id
JOIN booking_seats bs ON bs.booking_id=b.id
JOIN screening_seats ss ON ss.id=bs.screening_seat_id `

func (s *Store) List(ctx context.Context, userID string) ([]Booking, error) {
	return s.query(ctx, `WHERE b.user_id=$1 ORDER BY b.created_at DESC,ss.row_label,ss.seat_key`, userID)
}

func (s *Store) Get(ctx context.Context, bookingID, userID string) (Booking, error) {
	items, err := s.query(ctx, `WHERE b.id=$1 AND b.user_id=$2 ORDER BY ss.row_label,ss.seat_key`, bookingID, userID)
	if err != nil {
		return Booking{}, err
	}
	if len(items) == 0 {
		return Booking{}, ErrBookingNotFound
	}
	return items[0], nil
}

func (s *Store) query(ctx context.Context, where string, args ...any) ([]Booking, error) {
	rows, err := s.pool.Query(ctx, bookingSelect+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Booking{}
	positions := map[string]int{}
	for rows.Next() {
		var current Booking
		var bookedSeat seat.Seat
		if err := rows.Scan(&current.ID, &current.Reference, &current.ScreeningID, &current.Status, &current.MovieTitle, &current.VenueName, &current.StartsAt, &current.TotalPaise, &current.CancelledAt, &current.CustomerEmail, &bookedSeat.ID, &bookedSeat.Key, &bookedSeat.Row, &bookedSeat.Number, &bookedSeat.Category, &bookedSeat.PricePaise, &bookedSeat.Status); err != nil {
			return nil, err
		}
		position, exists := positions[current.ID]
		if !exists {
			current.Seats = []seat.Seat{}
			current.CanCancel = current.Status == "confirmed" && current.StartsAt.After(time.Now().Add(24*time.Hour))
			positions[current.ID] = len(items)
			items = append(items, current)
			position = len(items) - 1
		}
		items[position].Seats = append(items[position].Seats, bookedSeat)
	}
	return items, rows.Err()
}

func (s *Store) Cancel(ctx context.Context, bookingID, userID string) (Booking, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Booking{}, err
	}
	defer tx.Rollback(ctx)
	var status string
	var startsAt time.Time
	err = tx.QueryRow(ctx, `SELECT b.status::text,sc.starts_at FROM bookings b JOIN screenings sc ON sc.id=b.screening_id WHERE b.id=$1 AND b.user_id=$2 FOR UPDATE OF b`, bookingID, userID).Scan(&status, &startsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, ErrBookingNotFound
	}
	if err != nil {
		return Booking{}, err
	}
	if status != "confirmed" || !startsAt.After(time.Now().Add(24*time.Hour)) {
		return Booking{}, ErrCancellationClosed
	}
	if _, err = tx.Exec(ctx, `UPDATE bookings SET status='cancelled',cancelled_at=now() WHERE id=$1`, bookingID); err != nil {
		return Booking{}, err
	}
	command, err := tx.Exec(ctx, `UPDATE screening_seats ss SET status='available' FROM booking_seats bs WHERE bs.booking_id=$1 AND bs.screening_seat_id=ss.id AND ss.status='booked'`, bookingID)
	if err != nil {
		return Booking{}, err
	}
	if command.RowsAffected() == 0 {
		return Booking{}, fmt.Errorf("booking has no booked seats")
	}
	if err = tx.Commit(ctx); err != nil {
		return Booking{}, err
	}
	return s.Get(ctx, bookingID, userID)
}
