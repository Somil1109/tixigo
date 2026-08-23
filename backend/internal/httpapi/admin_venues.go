package httpapi

import (
	"encoding/json"
	"github.com/tixigo/tixigo-api/internal/venue"
	"net/http"
	"strings"
)

type venueHandler struct{ store *venue.Store }

func (h venueHandler) create(w http.ResponseWriter, r *http.Request) {
	var in venue.Venue
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Address) == "" || strings.TrimSpace(in.City) == "" {
		writeJSON(w, 400, map[string]string{"message": "Name, address, city, and layout are required."})
		return
	}
	if err := in.Layout.Validate(); err != nil {
		writeJSON(w, 400, map[string]string{"message": err.Error()})
		return
	}
	created, err := h.store.Create(r.Context(), in, accessClaims(r).Subject)
	if err != nil {
		writeJSON(w, 500, map[string]string{"message": "Could not create venue."})
		return
	}
	writeJSON(w, 201, map[string]any{"data": created})
}
func (h venueHandler) list(w http.ResponseWriter, r *http.Request) {
	venues, err := h.store.List(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]string{"message": "Could not load venues."})
		return
	}
	writeJSON(w, 200, map[string]any{"data": venues})
}
