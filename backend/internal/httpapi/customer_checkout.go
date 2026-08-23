package httpapi

import (
	"encoding/base64"
	"fmt"
	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/tixigo/tixigo-api/internal/booking"
	"github.com/tixigo/tixigo-api/internal/notification"
	"github.com/tixigo/tixigo-api/internal/seat"
	"html"
	"net/http"
	"strings"
)

type checkoutHandler struct {
	seats    *seat.Store
	bookings *booking.Store
	email    notification.EmailSender
}

func (h checkoutHandler) details(w http.ResponseWriter, r *http.Request) {
	result, err := h.seats.HoldDetails(r.Context(), chi.URLParam(r, "holdID"), accessClaims(r).Subject)
	if err != nil {
		writeJSON(w, 404, map[string]string{"message": "Hold was not found or has expired."})
		return
	}
	writeJSON(w, 200, map[string]any{"data": result})
}
func (h checkoutHandler) confirm(w http.ResponseWriter, r *http.Request) {
	result, err := h.bookings.Confirm(r.Context(), chi.URLParam(r, "holdID"), accessClaims(r).Subject)
	if err != nil {
		writeJSON(w, 409, map[string]string{"message": "Hold was not found or has expired."})
		return
	}
	png, err := qrcode.Encode(result.Reference, qrcode.Medium, 256)
	if err != nil {
		writeJSON(w, 500, map[string]string{"message": "Booking confirmed, but ticket generation failed."})
		return
	}
	qr := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	keys := make([]string, 0, len(result.Seats))
	for _, item := range result.Seats {
		keys = append(keys, item.Key)
	}
	body := fmt.Sprintf(`<h1>Your Tixigo ticket</h1><p><strong>%s</strong></p><p>%s · %s</p><p>Seats: %s</p><p>Booking reference: <strong>%s</strong></p><img alt="Ticket QR code" src="%s">`, html.EscapeString(result.MovieTitle), html.EscapeString(result.VenueName), result.StartsAt.Format("02 Jan 2006, 03:04 PM"), html.EscapeString(strings.Join(keys, ", ")), result.Reference, qr)
	emailSent := h.email.Send(r.Context(), result.CustomerEmail, "Your Tixigo booking "+result.Reference, body) == nil
	writeJSON(w, 201, map[string]any{"data": map[string]any{"booking": result, "qrCode": qr, "emailSent": emailSent}})
}
