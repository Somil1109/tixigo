package seat

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Seat struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	Row        string `json:"row"`
	Number     string `json:"number"`
	Category   string `json:"category"`
	PricePaise int    `json:"pricePaise"`
	Status     string `json:"status"`
}
type ScreeningMap struct {
	ScreeningID string    `json:"screeningId"`
	MovieID     string    `json:"movieId"`
	MovieTitle  string    `json:"movieTitle"`
	VenueName   string    `json:"venueName"`
	StartsAt    time.Time `json:"startsAt"`
	Seats       []Seat    `json:"seats"`
}
type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool} }

func (s *Store) Map(ctx context.Context, screeningID string) (ScreeningMap, error) {
	var result ScreeningMap
	err := s.pool.QueryRow(ctx, `SELECT sc.id::text,m.id::text,m.title,v.name,sc.starts_at FROM screenings sc JOIN movies m ON m.id=sc.movie_id JOIN venues v ON v.id=sc.venue_id WHERE sc.id=$1 AND sc.status='scheduled' AND m.status='published'`, screeningID).Scan(&result.ScreeningID, &result.MovieID, &result.MovieTitle, &result.VenueName, &result.StartsAt)
	if err != nil {
		return result, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,seat_key,row_label,seat_number,category,price_paise,CASE WHEN status='held' AND hold_expires_at<=now() THEN 'available' ELSE status::text END FROM screening_seats WHERE screening_id=$1 ORDER BY row_label,seat_key`, screeningID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	result.Seats = []Seat{}
	for rows.Next() {
		var item Seat
		if err := rows.Scan(&item.ID, &item.Key, &item.Row, &item.Number, &item.Category, &item.PricePaise, &item.Status); err != nil {
			return result, err
		}
		result.Seats = append(result.Seats, item)
	}
	return result, rows.Err()
}
