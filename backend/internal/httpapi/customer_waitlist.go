package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tixigo/tixigo-api/internal/notification"
	"github.com/tixigo/tixigo-api/internal/realtime"
	"github.com/tixigo/tixigo-api/internal/waitlist"
)

type waitlistHandler struct {
	waitlist *waitlist.Store
	email    notification.EmailSender
	hub      *realtime.Hub
}

func (h waitlistHandler) join(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Category string `json:"category"`
		Quantity int    `json:"quantity"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Category and quantity are required."})
		return
	}
	screeningID := chi.URLParam(r, "screeningID")
	entry, err := h.waitlist.Join(r.Context(), accessClaims(r).Subject, screeningID, input.Category, input.Quantity)
	if errors.Is(err, waitlist.ErrActiveEntry) {
		writeJSON(w, http.StatusConflict, map[string]string{"message": "You already have an active waitlist entry for this category."})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "This waitlist entry is not valid."})
		return
	}
	h.match(r, screeningID)
	writeJSON(w, http.StatusCreated, map[string]any{"data": entry})
}

func (h waitlistHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.waitlist.List(r.Context(), accessClaims(r).Subject)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Waitlist entries could not be loaded."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h waitlistHandler) cancel(w http.ResponseWriter, r *http.Request) {
	screeningID, err := h.waitlist.Cancel(r.Context(), chi.URLParam(r, "entryID"), accessClaims(r).Subject)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Active waitlist entry was not found."})
		return
	}
	h.hub.Publish(screeningID)
	h.match(r, screeningID)
	w.WriteHeader(http.StatusNoContent)
}

func (h waitlistHandler) match(r *http.Request, screeningID string) {
	offers, err := h.waitlist.Match(r.Context(), screeningID)
	if err != nil {
		return
	}
	if len(offers) > 0 {
		h.hub.Publish(screeningID)
	}
	waitlist.NotifyOffers(r.Context(), h.email, offers)
}
