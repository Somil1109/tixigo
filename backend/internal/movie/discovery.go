package movie

import (
	"context"
	"time"
)

type Filters struct {
	Query, City, Language, Genre string
	StartsAfter, StartsBefore    *time.Time
	MinPrice, MaxPrice           int
}
type PublicMovie struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description,omitempty"`
	PosterURL       string    `json:"posterUrl"`
	TrailerURL      *string   `json:"trailerUrl,omitempty"`
	Genres          []string  `json:"genres"`
	Language        string    `json:"language"`
	DurationMinutes int       `json:"durationMinutes"`
	AgeRating       string    `json:"ageRating"`
	NextShowtime    time.Time `json:"nextShowtime"`
	FromPrice       int       `json:"fromPrice"`
}
type PublicScreening struct {
	ID        string         `json:"id"`
	VenueID   string         `json:"venueId"`
	VenueName string         `json:"venueName"`
	City      string         `json:"city"`
	StartsAt  time.Time      `json:"startsAt"`
	Prices    map[string]int `json:"prices"`
}
type MovieDetails struct {
	PublicMovie
	Screenings []PublicScreening `json:"screenings"`
}

func (s *Store) Search(ctx context.Context, f Filters) ([]PublicMovie, error) {
	rows, err := s.pool.Query(ctx, `SELECT m.id::text,m.title,m.poster_url,m.genre,m.language,m.duration_minutes,m.age_rating,MIN(sc.starts_at),MIN(p.price_paise) FROM movies m JOIN screenings sc ON sc.movie_id=m.id JOIN venues v ON v.id=sc.venue_id JOIN screening_category_prices p ON p.screening_id=sc.id WHERE m.status='published' AND sc.status='scheduled' AND sc.starts_at>now() AND ($1='' OR m.title ILIKE '%'||$1||'%') AND ($2='' OR lower(v.city)=lower($2)) AND ($3='' OR lower(m.language)=lower($3)) AND ($4='' OR $4=ANY(m.genre)) AND ($5::timestamptz IS NULL OR sc.starts_at >= $5) AND ($6::timestamptz IS NULL OR sc.starts_at < $6) AND ($7=0 OR p.price_paise >= $7) AND ($8=0 OR p.price_paise <= $8) GROUP BY m.id ORDER BY MIN(sc.starts_at),m.title`, f.Query, f.City, f.Language, f.Genre, f.StartsAfter, f.StartsBefore, f.MinPrice, f.MaxPrice)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PublicMovie{}
	for rows.Next() {
		var item PublicMovie
		if err := rows.Scan(&item.ID, &item.Title, &item.PosterURL, &item.Genres, &item.Language, &item.DurationMinutes, &item.AgeRating, &item.NextShowtime, &item.FromPrice); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s *Store) Details(ctx context.Context, id string) (MovieDetails, error) {
	var out MovieDetails
	err := s.pool.QueryRow(ctx, `SELECT id::text,title,description,poster_url,trailer_url,genre,language,duration_minutes,age_rating FROM movies WHERE id=$1 AND status='published'`, id).Scan(&out.ID, &out.Title, &out.Description, &out.PosterURL, &out.TrailerURL, &out.Genres, &out.Language, &out.DurationMinutes, &out.AgeRating)
	if err != nil {
		return out, err
	}
	rows, err := s.pool.Query(ctx, `SELECT sc.id::text,v.id::text,v.name,v.city,sc.starts_at,p.category,p.price_paise FROM screenings sc JOIN venues v ON v.id=sc.venue_id JOIN screening_category_prices p ON p.screening_id=sc.id WHERE sc.movie_id=$1 AND sc.status='scheduled' AND sc.starts_at>now() ORDER BY sc.starts_at,p.category`, id)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	byID := map[string]int{}
	for rows.Next() {
		var id, venueID, name, city, category string
		var starts time.Time
		var price int
		if err := rows.Scan(&id, &venueID, &name, &city, &starts, &category, &price); err != nil {
			return out, err
		}
		index, ok := byID[id]
		if !ok {
			index = len(out.Screenings)
			byID[id] = index
			out.Screenings = append(out.Screenings, PublicScreening{ID: id, VenueID: venueID, VenueName: name, City: city, StartsAt: starts, Prices: map[string]int{}})
		}
		out.Screenings[index].Prices[category] = price
	}
	return out, rows.Err()
}
