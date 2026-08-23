package booking

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrTicketNotFound = errors.New("ticket was not found")
var ErrTicketInactive = errors.New("ticket is not active")
var ErrTicketUsed = errors.New("ticket has already been used")
var ErrOutsideAdmissionWindow = errors.New("ticket is outside its admission window")

type Admission struct {
	BookingID   string     `json:"bookingId"`
	Reference   string     `json:"reference"`
	MovieTitle  string     `json:"movieTitle"`
	VenueName   string     `json:"venueName"`
	StartsAt    time.Time  `json:"startsAt"`
	Status      string     `json:"status"`
	Seats       []string   `json:"seats"`
	CheckedInAt *time.Time `json:"checkedInAt,omitempty"`
}

func (s *Store) Admit(ctx context.Context, reference, actorID, actorRole string) (Admission, error) {
	var result Admission
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	var screeningStatus string
	var durationMinutes int
	err = tx.QueryRow(ctx, `SELECT b.id::text,b.reference,m.title,v.name,sc.starts_at,b.status::text,b.checked_in_at,sc.status::text,m.duration_minutes
FROM bookings b JOIN screenings sc ON sc.id=b.screening_id JOIN movies m ON m.id=sc.movie_id JOIN venues v ON v.id=sc.venue_id
WHERE b.reference=$1 AND ($3='admin' OR m.organiser_id=$2) FOR UPDATE OF b`, reference, actorID, actorRole).Scan(&result.BookingID, &result.Reference, &result.MovieTitle, &result.VenueName, &result.StartsAt, &result.Status, &result.CheckedInAt, &screeningStatus, &durationMinutes)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrTicketNotFound
	}
	if err != nil {
		return result, err
	}
	if result.Status != "confirmed" || screeningStatus != "scheduled" {
		return result, ErrTicketInactive
	}
	if result.CheckedInAt != nil {
		return result, ErrTicketUsed
	}
	now := time.Now()
	if now.Before(result.StartsAt.Add(-3*time.Hour)) || now.After(result.StartsAt.Add(time.Duration(durationMinutes+120)*time.Minute)) {
		return result, ErrOutsideAdmissionWindow
	}
	rows, err := tx.Query(ctx, `SELECT ss.seat_key FROM booking_seats bs JOIN screening_seats ss ON ss.id=bs.screening_seat_id WHERE bs.booking_id=$1 ORDER BY ss.row_label,ss.seat_key`, result.BookingID)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return result, err
		}
		result.Seats = append(result.Seats, key)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return result, err
	}
	if err = tx.QueryRow(ctx, `UPDATE bookings SET checked_in_at=now(),checked_in_by=$2 WHERE id=$1 RETURNING checked_in_at`, result.BookingID, actorID).Scan(&result.CheckedInAt); err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return result, err
	}
	return result, nil
}
