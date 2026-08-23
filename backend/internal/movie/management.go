package movie

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tixigo/tixigo-api/internal/venue"
)

var ErrNotEditable = errors.New("movie or screening cannot be edited")

type ScreeningOverview struct {
	ID           string    `json:"id"`
	VenueName    string    `json:"venueName"`
	StartsAt     time.Time `json:"startsAt"`
	Status       string    `json:"status"`
	TotalSeats   int       `json:"totalSeats"`
	BookedSeats  int       `json:"bookedSeats"`
	ActiveHolds  int       `json:"activeHolds"`
	BookingCount int       `json:"bookingCount"`
	RevenuePaise int       `json:"revenuePaise"`
}

type ManagedMovie struct {
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	Description     string              `json:"description"`
	PosterURL       string              `json:"posterUrl"`
	TrailerURL      *string             `json:"trailerUrl"`
	Genres          []string            `json:"genres"`
	Language        string              `json:"language"`
	DurationMinutes int                 `json:"durationMinutes"`
	AgeRating       string              `json:"ageRating"`
	Status          string              `json:"status"`
	Screenings      []ScreeningOverview `json:"screenings"`
}

type CancellationNotice struct {
	Email, Reference, MovieTitle, VenueName string
	StartsAt                                time.Time
}

func (s *Store) Managed(ctx context.Context, organiserID string) ([]ManagedMovie, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text,title,description,poster_url,trailer_url,genre,language,duration_minutes,age_rating,status::text FROM movies WHERE organiser_id=$1 ORDER BY created_at DESC`, organiserID)
	if err != nil {
		return nil, err
	}
	items := []ManagedMovie{}
	positions := map[string]int{}
	for rows.Next() {
		var item ManagedMovie
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.PosterURL, &item.TrailerURL, &item.Genres, &item.Language, &item.DurationMinutes, &item.AgeRating, &item.Status); err != nil {
			rows.Close()
			return nil, err
		}
		item.Screenings = []ScreeningOverview{}
		positions[item.ID] = len(items)
		items = append(items, item)
	}
	rows.Close()
	if len(items) == 0 {
		return items, nil
	}
	screenings, err := s.pool.Query(ctx, `SELECT sc.movie_id::text,sc.id::text,v.name,sc.starts_at,sc.status::text,
	COUNT(ss.id),COUNT(ss.id) FILTER (WHERE ss.status='booked'),COUNT(ss.id) FILTER (WHERE ss.status IN ('held','waitlist_reserved')),
(SELECT COUNT(*) FROM bookings b WHERE b.screening_id=sc.id AND b.status='confirmed'),(SELECT COALESCE(SUM(b.total_paise),0) FROM bookings b WHERE b.screening_id=sc.id AND b.status='confirmed')
FROM screenings sc JOIN movies m ON m.id=sc.movie_id JOIN venues v ON v.id=sc.venue_id LEFT JOIN screening_seats ss ON ss.screening_id=sc.id
WHERE m.organiser_id=$1 GROUP BY sc.id,v.name ORDER BY sc.starts_at DESC`, organiserID)
	if err != nil {
		return nil, err
	}
	defer screenings.Close()
	for screenings.Next() {
		var movieID string
		var item ScreeningOverview
		if err := screenings.Scan(&movieID, &item.ID, &item.VenueName, &item.StartsAt, &item.Status, &item.TotalSeats, &item.BookedSeats, &item.ActiveHolds, &item.BookingCount, &item.RevenuePaise); err != nil {
			return nil, err
		}
		if position, ok := positions[movieID]; ok {
			items[position].Screenings = append(items[position].Screenings, item)
		}
	}
	return items, screenings.Err()
}

func (s *Store) UpdateManaged(ctx context.Context, movieID, organiserID string, input ManagedMovie) error {
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.PosterURL) == "" || strings.TrimSpace(input.Language) == "" || strings.TrimSpace(input.AgeRating) == "" || input.DurationMinutes < 1 {
		return ErrNotEditable
	}
	command, err := s.pool.Exec(ctx, `UPDATE movies SET title=$3,description=$4,poster_url=$5,trailer_url=$6,genre=$7,language=$8,duration_minutes=$9,age_rating=$10,updated_at=now() WHERE id=$1 AND organiser_id=$2 AND status IN ('draft','rejected')`, movieID, organiserID, input.Title, input.Description, input.PosterURL, input.TrailerURL, input.Genres, input.Language, input.DurationMinutes, input.AgeRating)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotEditable
	}
	return nil
}

func (s *Store) AddScreening(ctx context.Context, movieID, organiserID string, input ScreeningInput) (string, error) {
	if input.VenueID == "" || input.StartsAt.Before(time.Now()) || len(input.Prices) == 0 {
		return "", ErrNotEditable
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var owned bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM movies WHERE id=$1 AND organiser_id=$2 AND status<>'pending_approval')`, movieID, organiserID).Scan(&owned); err != nil || !owned {
		return "", ErrNotEditable
	}
	var raw []byte
	if err = tx.QueryRow(ctx, `SELECT layout FROM venues WHERE id=$1`, input.VenueID).Scan(&raw); err != nil {
		return "", err
	}
	var layout venue.Layout
	if err = json.Unmarshal(raw, &layout); err != nil {
		return "", err
	}
	if err = layout.Validate(); err != nil {
		return "", err
	}
	for _, category := range layout.Categories {
		if input.Prices[category.Key] < 1 {
			return "", fmt.Errorf("missing positive price for category %s", category.Key)
		}
	}
	var screeningID string
	if err = tx.QueryRow(ctx, `INSERT INTO screenings(movie_id,venue_id,starts_at) VALUES($1,$2,$3) RETURNING id::text`, movieID, input.VenueID, input.StartsAt).Scan(&screeningID); err != nil {
		return "", err
	}
	for category, price := range input.Prices {
		if _, err = tx.Exec(ctx, `INSERT INTO screening_category_prices(screening_id,category,price_paise) VALUES($1,$2,$3)`, screeningID, category, price); err != nil {
			return "", err
		}
	}
	for _, row := range layout.Rows {
		for _, item := range row.Seats {
			if _, err = tx.Exec(ctx, `INSERT INTO screening_seats(screening_id,seat_key,row_label,seat_number,category,price_paise) VALUES($1,$2,$3,$4,$5,$6)`, screeningID, row.Label+item.Number, row.Label, item.Number, item.Category, input.Prices[item.Category]); err != nil {
				return "", err
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return screeningID, nil
}

func (s *Store) CancelScreening(ctx context.Context, screeningID, organiserID string) ([]CancellationNotice, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	command, err := tx.Exec(ctx, `UPDATE screenings sc SET status='cancelled',cancelled_at=now() FROM movies m WHERE sc.id=$1 AND sc.movie_id=m.id AND m.organiser_id=$2 AND sc.status='scheduled' AND sc.starts_at>now()`, screeningID, organiserID)
	if err != nil {
		return nil, err
	}
	if command.RowsAffected() == 0 {
		return nil, ErrNotEditable
	}
	rows, err := tx.Query(ctx, `SELECT u.email,b.reference,m.title,v.name,sc.starts_at FROM bookings b JOIN users u ON u.id=b.user_id JOIN screenings sc ON sc.id=b.screening_id JOIN movies m ON m.id=sc.movie_id JOIN venues v ON v.id=sc.venue_id WHERE b.screening_id=$1 AND b.status='confirmed' FOR UPDATE OF b`, screeningID)
	if err != nil {
		return nil, err
	}
	notices := []CancellationNotice{}
	for rows.Next() {
		var notice CancellationNotice
		if err := rows.Scan(&notice.Email, &notice.Reference, &notice.MovieTitle, &notice.VenueName, &notice.StartsAt); err != nil {
			rows.Close()
			return nil, err
		}
		notices = append(notices, notice)
	}
	rows.Close()
	if _, err = tx.Exec(ctx, `UPDATE bookings SET status='cancelled',cancelled_at=now() WHERE screening_id=$1 AND status='confirmed'`, screeningID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE waitlist_entries SET status='cancelled' WHERE screening_id=$1 AND status IN ('waiting','offered')`, screeningID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE seat_holds SET status='cancelled' WHERE screening_id=$1 AND status='active'`, screeningID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE screening_seats SET status='available',held_by=NULL,hold_id=NULL,hold_expires_at=NULL,waitlist_entry_id=NULL WHERE screening_id=$1`, screeningID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return notices, nil
}
