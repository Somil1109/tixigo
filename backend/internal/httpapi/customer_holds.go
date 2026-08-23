package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/tixigo/tixigo-api/internal/seat"
	"net/http"
)

type holdHandler struct{ seats *seat.Store }

func (h holdHandler) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SeatIDs []string `json:"seatIds"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		writeJSON(w, 400, map[string]string{"message": "Seat IDs are required."})
		return
	}
	hold, err := h.seats.Hold(r.Context(), chi.URLParam(r, "screeningID"), accessClaims(r).Subject, in.SeatIDs)
	if errors.Is(err, seat.ErrEmailUnverified) {
		writeJSON(w, 403, map[string]string{"message": "Verify your email before holding seats."})
		return
	}
	if err != nil {
		writeJSON(w, 409, map[string]string{"message": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]any{"data": hold})
}
func (h holdHandler) release(w http.ResponseWriter, r *http.Request) {
	if err := h.seats.Release(r.Context(), chi.URLParam(r, "holdID"), accessClaims(r).Subject); err != nil {
		writeJSON(w, 404, map[string]string{"message": "Active hold was not found."})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
