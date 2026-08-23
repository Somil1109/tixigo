package httpapi

import (
	"errors"
	"fmt"
	"html"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tixigo/tixigo-api/internal/booking"
	"github.com/tixigo/tixigo-api/internal/notification"
)

type bookingHandler struct {
	bookings *booking.Store
	email    notification.EmailSender
}

func (h bookingHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.bookings.List(r.Context(), accessClaims(r).Subject)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Bookings could not be loaded."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h bookingHandler) details(w http.ResponseWriter, r *http.Request) {
	result, err := h.bookings.Get(r.Context(), chi.URLParam(r, "bookingID"), accessClaims(r).Subject)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Booking was not found."})
		return
	}
	qr, err := ticketQRCode(result.Reference)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Ticket could not be generated."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"booking": result, "qrCode": qr}})
}

func (h bookingHandler) cancel(w http.ResponseWriter, r *http.Request) {
	result, err := h.bookings.Cancel(r.Context(), chi.URLParam(r, "bookingID"), accessClaims(r).Subject)
	if errors.Is(err, booking.ErrBookingNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Booking was not found."})
		return
	}
	if errors.Is(err, booking.ErrCancellationClosed) {
		writeJSON(w, http.StatusConflict, map[string]string{"message": "Bookings can only be cancelled more than 24 hours before showtime."})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Booking could not be cancelled."})
		return
	}
	body := fmt.Sprintf(`<h1>Your Tixigo booking was cancelled</h1><p><strong>%s</strong></p><p>%s · %s</p><p>Booking reference: <strong>%s</strong></p>`, html.EscapeString(result.MovieTitle), html.EscapeString(result.VenueName), result.StartsAt.Format("02 Jan 2006, 03:04 PM"), result.Reference)
	emailSent := h.email.Send(r.Context(), result.CustomerEmail, "Tixigo cancellation "+result.Reference, body) == nil
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"booking": result, "emailSent": emailSent}})
}
