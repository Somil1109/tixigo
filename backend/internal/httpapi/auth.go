package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tixigo/tixigo-api/internal/auth"
	"github.com/tixigo/tixigo-api/internal/notification"
)

type authHandler struct {
	users         *auth.UserStore
	sessions      *auth.SessionStore
	accountTokens *auth.AccountTokenStore
	tokens        *auth.TokenManager
	email         notification.EmailSender
	webOrigin     string
}

func newAuthHandler(users *auth.UserStore, sessions *auth.SessionStore, accountTokens *auth.AccountTokenStore, tokens *auth.TokenManager, email notification.EmailSender, webOrigin string) authHandler {
	return authHandler{users: users, sessions: sessions, accountTokens: accountTokens, tokens: tokens, email: email, webOrigin: webOrigin}
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
	rawToken, tokenHash, err := auth.NewAccountToken()
	if err != nil || h.accountTokens.Create(r.Context(), u.ID, "verify_email", tokenHash, time.Now().Add(24*time.Hour)) != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Could not create verification request."})
		return
	}
	verificationURL := h.webOrigin + "/verify-email?token=" + url.QueryEscape(rawToken)
	if err := h.email.Send(r.Context(), u.Email, "Verify your Tixigo email", fmt.Sprintf(`<p>Welcome to Tixigo.</p><p><a href="%s">Verify your email</a></p>`, verificationURL)); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"message": "Account created, but verification email could not be sent."})
		return
	}
	writeJSON(w, 201, map[string]any{"data": map[string]any{"id": u.ID, "email": u.Email, "fullName": u.FullName, "role": u.Role}})
}

func (h authHandler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Verification token is required."})
		return
	}
	if h.accountTokens.VerifyEmail(r.Context(), auth.HashAccountToken(in.Token)) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Verification link is invalid or expired."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Email verified successfully."})
}

func (h authHandler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if json.NewDecoder(r.Body).Decode(&in) == nil {
		if u, _, err := h.users.ByEmail(r.Context(), in.Email); err == nil {
			rawToken, hash, tokenErr := auth.NewAccountToken()
			if tokenErr == nil {
				if h.accountTokens.Create(r.Context(), u.ID, "reset_password", hash, time.Now().Add(time.Hour)) == nil {
					resetURL := h.webOrigin + "/reset-password?token=" + url.QueryEscape(rawToken)
					_ = h.email.Send(r.Context(), u.Email, "Reset your Tixigo password", fmt.Sprintf(`<p><a href="%s">Reset your password</a>. This link expires in one hour.</p>`, resetURL))
				}
			}
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "If that account exists, a reset link will be sent."})
}

func (h authHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Token and new password are required."})
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	if h.accountTokens.ResetPassword(r.Context(), auth.HashAccountToken(in.Token), hash) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Reset link is invalid or expired."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Password updated. Please sign in again."})
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

func (h authHandler) me(w http.ResponseWriter, r *http.Request) {
	u, err := h.users.ByID(r.Context(), accessClaims(r).Subject)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Account unavailable."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": u})
}
