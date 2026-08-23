package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tixigo/tixigo-api/internal/auth"
)

type adminUserHandler struct{ users *auth.UserStore }

func (h adminUserHandler) updateRole(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Role auth.Role `json:"role"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || (in.Role != auth.RoleCustomer && in.Role != auth.RoleOrganiser) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Role must be customer or organiser."})
		return
	}
	u, err := h.users.UpdateRole(r.Context(), chi.URLParam(r, "userID"), in.Role)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "User was not found or cannot be updated."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": u})
}
