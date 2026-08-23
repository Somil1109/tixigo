package httpapi

import (
	"encoding/json"
	"github.com/tixigo/tixigo-api/internal/auth"
	"net/http"
	"strings"
	"time"
)

type authHandler struct {
	users    *auth.UserStore
	sessions *auth.SessionStore
	tokens   *auth.TokenManager
}

func newAuthHandler(users *auth.UserStore, sessions *auth.SessionStore, tokens *auth.TokenManager) authHandler {
	return authHandler{users, sessions, tokens}
}
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

func (h authHandler) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		writeJSON(w, 400, map[string]string{"message": "Email and password are required."})
		return
	}
	u, hash, err := h.users.ByEmail(r.Context(), in.Email)
	if err != nil || auth.ComparePassword(hash, in.Password) != nil {
		writeJSON(w, 401, map[string]string{"message": "Invalid email or password."})
		return
	}
	token, err := h.tokens.NewAccessToken(u.ID, u.Role, time.Now())
	if err != nil {
		writeJSON(w, 500, map[string]string{"message": "Could not sign in."})
		return
	}
	raw, hash, expiry, err := h.tokens.NewRefreshToken(time.Now())
	if err != nil || h.sessions.Create(r.Context(), u.ID, hash, expiry) != nil {
		writeJSON(w, 500, map[string]string{"message": "Could not create session."})
		return
	}
	h.setRefreshCookie(w, raw, expiry)
	writeJSON(w, 200, map[string]any{"data": map[string]any{"accessToken": token, "user": u}})
}
func (h authHandler) refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("tixigo_refresh")
	if err != nil {
		writeJSON(w, 401, map[string]string{"message": "Session expired."})
		return
	}
	id, err := h.sessions.UserID(r.Context(), h.tokens.HashRefreshToken(c.Value))
	if err != nil {
		writeJSON(w, 401, map[string]string{"message": "Session expired."})
		return
	}
	_ = h.sessions.Revoke(r.Context(), h.tokens.HashRefreshToken(c.Value))
	u, err := h.users.ByID(r.Context(), id)
	if err != nil {
		writeJSON(w, 401, map[string]string{"message": "Session expired."})
		return
	}
	raw, hash, expiry, _ := h.tokens.NewRefreshToken(time.Now())
	if h.sessions.Create(r.Context(), id, hash, expiry) != nil {
		writeJSON(w, 500, map[string]string{"message": "Could not refresh session."})
		return
	}
	h.setRefreshCookie(w, raw, expiry)
	token, _ := h.tokens.NewAccessToken(u.ID, u.Role, time.Now())
	writeJSON(w, 200, map[string]any{"data": map[string]any{"accessToken": token, "user": u}})
}
func (h authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("tixigo_refresh"); err == nil {
		_ = h.sessions.Revoke(r.Context(), h.tokens.HashRefreshToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: "tixigo_refresh", Value: "", Path: "/api/v1/auth", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}
func (h authHandler) setRefreshCookie(w http.ResponseWriter, value string, expiry time.Time) {
	http.SetCookie(w, &http.Cookie{Name: "tixigo_refresh", Value: value, Path: "/api/v1/auth", Expires: expiry, HttpOnly: true, Secure: false, SameSite: http.SameSiteLaxMode})
}
