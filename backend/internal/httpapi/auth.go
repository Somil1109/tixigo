package httpapi

import (
	"encoding/json"
	"github.com/tixigo/tixigo-api/internal/auth"
	"net/http"
	"strings"
)

type authHandler struct{ users *auth.UserStore }

func newAuthHandler(users *auth.UserStore) authHandler { return authHandler{users} }
func (h authHandler) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		FullName string `json:"fullName"`
		Password string `json:"password"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || !strings.Contains(in.Email, "@") || strings.TrimSpace(in.FullName) == "" {
		writeJSON(w, 400, map[string]string{"message": "Name, email, and password are required."})
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeJSON(w, 400, map[string]string{"message": err.Error()})
		return
	}
	u, err := h.users.Create(r.Context(), in.Email, in.FullName, hash)
	if err != nil {
		writeJSON(w, 409, map[string]string{"message": "An account with that email already exists."})
		return
	}
	writeJSON(w, 201, map[string]any{"data": map[string]any{"id": u.ID, "email": u.Email, "fullName": u.FullName, "role": u.Role}})
}
