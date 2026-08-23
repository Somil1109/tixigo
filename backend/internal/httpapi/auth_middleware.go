package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/tixigo/tixigo-api/internal/auth"
)

type accessClaimsContextKey struct{}

func requireAuth(tokens *auth.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme, rawToken, ok := strings.Cut(r.Header.Get("Authorization"), " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(rawToken) == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Authentication required."})
				return
			}

			claims, err := tokens.ParseAccessToken(strings.TrimSpace(rawToken))
			if err != nil || claims.Subject == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Invalid or expired access token."})
				return
			}

			ctx := context.WithValue(r.Context(), accessClaimsContextKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func accessClaims(r *http.Request) auth.AccessClaims {
	claims, _ := r.Context().Value(accessClaimsContextKey{}).(auth.AccessClaims)
	return claims
}
