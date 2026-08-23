package movie

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tixigo/tixigo-api/internal/venue"
)

type ScreeningInput struct {
	VenueID  string         `json:"venueId"`
	StartsAt time.Time      `json:"startsAt"`
	Prices   map[string]int `json:"prices"`
}
type Draft struct {
	ID              string           `json:"id"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	PosterURL       string           `json:"posterUrl"`
	TrailerURL      *string          `json:"trailerUrl"`
	Genres          []string         `json:"genres"`
	Language        string           `json:"language"`
	DurationMinutes int              `json:"durationMinutes"`
	AgeRating       string           `json:"ageRating"`
	Status          string           `json:"status"`
	Screenings      []ScreeningInput `json:"screenings"`
}
type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool} }

func (d Draft) Validate() error {
	if strings.TrimSpace(d.Title) == "" || strings.TrimSpace(d.Description) == "" || strings.TrimSpace(d.PosterURL) == "" || strings.TrimSpace(d.Language) == "" || strings.TrimSpace(d.AgeRating) == "" {
		return errors.New("movie details are incomplete")
	}
	if d.DurationMinutes < 1 || len(d.Screenings) == 0 {
		return errors.New("duration and at least one screening are required")
	}
	for _, s := range d.Screenings {
		if s.VenueID == "" || s.StartsAt.Before(time.Now()) || len(s.Prices) == 0 {
			return errors.New("each screening requires venue, future start time, and prices")
		}
	}
	return nil
}
func (s *Store) CreateDraft(ctx context.Context, d Draft, organiserID string) (Draft, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return d, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `INSERT INTO movies(title,description,poster_url,trailer_url,genre,language,duration_minutes,age_rating,organiser_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id::text,status`, d.Title, d.Description, d.PosterURL, d.TrailerURL, d.Genres, d.Language, d.DurationMinutes, d.AgeRating, organiserID).Scan(&d.ID, &d.Status)
	if err != nil {
		return d, err
	}
	for _, screening := range d.Screenings {
		var raw []byte
		if err = tx.QueryRow(ctx, `SELECT layout FROM venues WHERE id=$1`, screening.VenueID).Scan(&raw); err != nil {
			return d, err
		}
		var layout venue.Layout
		if err = json.Unmarshal(raw, &layout); err != nil {
			return d, err
		}
		if err = layout.Validate(); err != nil {
			return d, err
		}
		for _, category := range layout.Categories {
			price, ok := screening.Prices[category.Key]
			if !ok || price < 1 {
				return d, fmt.Errorf("missing positive price for category %s", category.Key)
			}
		}
		var screeningID string
		if err = tx.QueryRow(ctx, `INSERT INTO screenings(movie_id,venue_id,starts_at) VALUES($1,$2,$3) RETURNING id::text`, d.ID, screening.VenueID, screening.StartsAt).Scan(&screeningID); err != nil {
			return d, err
		}
		for category, price := range screening.Prices {
			if _, err = tx.Exec(ctx, `INSERT INTO screening_category_prices(screening_id,category,price_paise) VALUES($1,$2,$3)`, screeningID, category, price); err != nil {
				return d, err
			}
		}
		for _, row := range layout.Rows {
			for _, seat := range row.Seats {
				price := screening.Prices[seat.Category]
				key := row.Label + seat.Number
				if _, err = tx.Exec(ctx, `INSERT INTO screening_seats(screening_id,seat_key,row_label,seat_number,category,price_paise) VALUES($1,$2,$3,$4,$5,$6)`, screeningID, key, row.Label, seat.Number, seat.Category, price); err != nil {
					return d, err
				}
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return d, err
	}
	return d, nil
}
