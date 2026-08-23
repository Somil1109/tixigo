package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tixigo/tixigo-api/internal/auth"
)

func TestRequireAuth(t *testing.T) {
	tokens, err := auth.NewTokenManager(strings.Repeat("a", 32), strings.Repeat("b", 32), time.Minute, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tokens.NewAccessToken("user-123", auth.RoleCustomer, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	handler := requireAuth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if accessClaims(r).Subject != "user-123" {
			t.Fatal("missing authenticated user")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d", res.Code)
	}
}

func TestRequireRoleRejectsWrongRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), accessClaimsContextKey{}, auth.AccessClaims{Role: auth.RoleCustomer}))
	res := httptest.NewRecorder()
	requireRole(auth.RoleAdmin)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler should not run") })).ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d", res.Code)
	}
}
