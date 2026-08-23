package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/tixigo/tixigo-api/internal/booking"
)

func (h bookingHandler) admit(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reference string `json:"reference"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil || strings.TrimSpace(input.Reference) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Ticket reference is required."})
		return
	}
	claims := accessClaims(r)
	result, err := h.bookings.Admit(r.Context(), strings.ToUpper(strings.TrimSpace(input.Reference)), claims.Subject, string(claims.Role))
	status := http.StatusConflict
	message := "Ticket could not be admitted."
	switch {
	case errors.Is(err, booking.ErrTicketNotFound):
		status, message = http.StatusNotFound, "Ticket was not found or does not belong to your screening."
	case errors.Is(err, booking.ErrTicketInactive):
		message = "This booking or screening is cancelled."
	case errors.Is(err, booking.ErrTicketUsed):
		message = "This ticket has already been used."
	case errors.Is(err, booking.ErrOutsideAdmissionWindow):
		message = "Admission opens three hours before the screening."
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]any{"data": result})
		return
	}
	writeJSON(w, status, map[string]string{"message": message})
}
