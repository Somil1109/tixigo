package httpapi

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tixigo/tixigo-api/internal/movie"
)

func (h organiserMovieHandler) managed(w http.ResponseWriter, r *http.Request) {
	items, err := h.movies.Managed(r.Context(), accessClaims(r).Subject)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Movies could not be loaded."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h organiserMovieHandler) update(w http.ResponseWriter, r *http.Request) {
	var input movie.ManagedMovie
	if json.NewDecoder(r.Body).Decode(&input) != nil || h.movies.UpdateManaged(r.Context(), chi.URLParam(r, "movieID"), accessClaims(r).Subject, input) != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"message": "Only complete draft or rejected movies can be edited."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Movie updated."})
}

func (h organiserMovieHandler) addScreening(w http.ResponseWriter, r *http.Request) {
	var input movie.ScreeningInput
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid screening."})
		return
	}
	id, err := h.movies.AddScreening(r.Context(), chi.URLParam(r, "movieID"), accessClaims(r).Subject, input)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]string{"id": id}})
}

func (h organiserMovieHandler) cancelScreening(w http.ResponseWriter, r *http.Request) {
	notices, err := h.movies.CancelScreening(r.Context(), chi.URLParam(r, "screeningID"), accessClaims(r).Subject)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"message": "Screening cannot be cancelled."})
		return
	}
	for _, notice := range notices {
		body := fmt.Sprintf(`<h1>Your screening was cancelled</h1><p><strong>%s</strong> at %s on %s will not go ahead.</p><p>Booking %s has been cancelled.</p>`, html.EscapeString(notice.MovieTitle), html.EscapeString(notice.VenueName), notice.StartsAt.Format("02 Jan 2006, 03:04 PM"), notice.Reference)
		_ = h.email.Send(r.Context(), notice.Email, "Tixigo screening cancellation", body)
	}
	h.hub.Publish(chi.URLParam(r, "screeningID"))
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]int{"cancelledBookings": len(notices)}})
}
