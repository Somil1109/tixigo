package httpapi

import (
	"github.com/go-chi/chi/v5"
	"github.com/tixigo/tixigo-api/internal/seat"
	"net/http"
)

type publicSeatHandler struct{ seats *seat.Store }

func (h publicSeatHandler) seatMap(w http.ResponseWriter, r *http.Request) {
	result, err := h.seats.Map(r.Context(), chi.URLParam(r, "screeningID"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Screening was not found."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}
