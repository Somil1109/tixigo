package httpapi

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/tixigo/tixigo-api/internal/movie"
	"net/http"
	"strings"
)

type adminMovieHandler struct{ movies *movie.Store }

func (h adminMovieHandler) pending(w http.ResponseWriter, r *http.Request) {
	items, err := h.movies.Pending(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]string{"message": "Could not load pending movies."})
		return
	}
	writeJSON(w, 200, map[string]any{"data": items})
}
func (h adminMovieHandler) review(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || (in.Decision != "approve" && in.Decision != "reject") {
		writeJSON(w, 400, map[string]string{"message": "Decision must be approve or reject."})
		return
	}
	status, reason := "published", ""
	if in.Decision == "reject" {
		status, reason = "rejected", strings.TrimSpace(in.Reason)
		if reason == "" {
			writeJSON(w, 400, map[string]string{"message": "A rejection reason is required."})
			return
		}
	}
	if err := h.movies.Review(r.Context(), chi.URLParam(r, "movieID"), accessClaims(r).Subject, status, reason); err != nil {
		writeJSON(w, 409, map[string]string{"message": "Movie is not pending approval."})
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Movie review completed."})
}
