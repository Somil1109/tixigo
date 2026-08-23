package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tixigo/tixigo-api/internal/auth"
	"github.com/tixigo/tixigo-api/internal/booking"
	"github.com/tixigo/tixigo-api/internal/config"
	"github.com/tixigo/tixigo-api/internal/media"
	"github.com/tixigo/tixigo-api/internal/movie"
	"github.com/tixigo/tixigo-api/internal/notification"
	"github.com/tixigo/tixigo-api/internal/seat"
	"github.com/tixigo/tixigo-api/internal/venue"
)

func NewRouter(cfg config.Config, pool *pgxpool.Pool, tokens *auth.TokenManager, email notification.EmailSender) http.Handler {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.WebOrigin},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, request *http.Request) {
		pingCtx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Route("/api/v1", func(r chi.Router) {
		h := newAuthHandler(auth.NewUserStore(pool), auth.NewSessionStore(pool), auth.NewAccountTokenStore(pool), tokens, email, cfg.WebOrigin)
		r.Post("/auth/register", h.register)
		r.Post("/auth/login", h.login)
		r.Post("/auth/refresh", h.refresh)
		r.Post("/auth/logout", h.logout)
		r.Post("/auth/verify-email", h.verifyEmail)
		r.Post("/auth/forgot-password", h.forgotPassword)
		r.Post("/auth/reset-password", h.resetPassword)
		r.With(requireAuth(tokens)).Get("/auth/me", h.me)
		r.Route("/admin", func(r chi.Router) {
			r.Use(requireAuth(tokens))
			r.Use(requireRole(auth.RoleAdmin))
			users := adminUserHandler{auth.NewUserStore(pool)}
			r.Get("/users", users.list)
			r.Patch("/users/{userID}/role", users.updateRole)
			venues := venueHandler{venue.NewStore(pool)}
			r.Post("/venues", venues.create)
			r.Get("/venues", venues.list)
			movies := adminMovieHandler{movie.NewStore(pool)}
			r.Get("/movies/pending", movies.pending)
			r.Patch("/movies/{movieID}/review", movies.review)
		})
		r.Route("/organiser", func(r chi.Router) {
			r.Use(requireAuth(tokens))
			r.Use(requireRole(auth.RoleOrganiser, auth.RoleAdmin))
			h := organiserMovieHandler{movie.NewStore(pool), media.NewCloudinary(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)}
			r.Post("/movies", h.create)
			r.Post("/movies/{movieID}/submit", h.submit)
			r.Post("/media/posters", h.uploadPoster)
			r.Get("/venues", venueHandler{venue.NewStore(pool)}.list)
		})
		publicMovies := publicMovieHandler{movie.NewStore(pool)}
		r.Get("/movies", publicMovies.list)
		r.Get("/movies/{movieID}", publicMovies.details)
		r.Get("/screenings/{screeningID}/seats", publicSeatHandler{seat.NewStore(pool)}.seatMap)
		holds := holdHandler{seat.NewStore(pool)}
		r.With(requireAuth(tokens), requireRole(auth.RoleCustomer)).Post("/screenings/{screeningID}/holds", holds.create)
		r.With(requireAuth(tokens), requireRole(auth.RoleCustomer)).Delete("/holds/{holdID}", holds.release)
		checkout := checkoutHandler{seat.NewStore(pool), booking.NewStore(pool), email}
		r.With(requireAuth(tokens), requireRole(auth.RoleCustomer)).Get("/holds/{holdID}", checkout.details)
		r.With(requireAuth(tokens), requireRole(auth.RoleCustomer)).Post("/holds/{holdID}/checkout", checkout.confirm)
		bookings := bookingHandler{booking.NewStore(pool), email}
		r.With(requireAuth(tokens), requireRole(auth.RoleCustomer)).Get("/bookings", bookings.list)
		r.With(requireAuth(tokens), requireRole(auth.RoleCustomer)).Get("/bookings/{bookingID}", bookings.details)
		r.With(requireAuth(tokens), requireRole(auth.RoleCustomer)).Post("/bookings/{bookingID}/cancel", bookings.cancel)
	})
	return r
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
