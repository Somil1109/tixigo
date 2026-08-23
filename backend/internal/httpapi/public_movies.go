package httpapi

import (
	"github.com/go-chi/chi/v5"
	"github.com/tixigo/tixigo-api/internal/movie"
	"net/http"
	"strconv"
	"time"
)

type publicMovieHandler struct{ movies *movie.Store }

func parsePrice(value string) int {
	parsed, _ := strconv.Atoi(value)
	if parsed < 0 {
		return 0
	}
	return parsed
}
func indiaDate(value string) (*time.Time, *time.Time, error) {
	if value == "" {
		return nil, nil, nil
	}
	location, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return nil, nil, err
	}
	start, err := time.ParseInLocation("2006-01-02", value, location)
	if err != nil {
		return nil, nil, err
	}
	end := start.AddDate(0, 0, 1)
	return &start, &end, nil
}
func (h publicMovieHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	after, before, err := indiaDate(q.Get("date"))
	if err != nil {
		writeJSON(w, 400, map[string]string{"message": "Date must use YYYY-MM-DD."})
		return
	}
	items, err := h.movies.Search(r.Context(), movie.Filters{Query: q.Get("q"), City: q.Get("city"), Language: q.Get("language"), Genre: q.Get("genre"), StartsAfter: after, StartsBefore: before, MinPrice: parsePrice(q.Get("minPrice")), MaxPrice: parsePrice(q.Get("maxPrice"))})
	if err != nil {
		writeJSON(w, 500, map[string]string{"message": "Could not load movies."})
		return
	}
	writeJSON(w, 200, map[string]any{"data": items})
}
func (h publicMovieHandler) details(w http.ResponseWriter, r *http.Request) {
	item, err := h.movies.Details(r.Context(), chi.URLParam(r, "movieID"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"message": "Movie was not found."})
		return
	}
	writeJSON(w, 200, map[string]any{"data": item})
}
