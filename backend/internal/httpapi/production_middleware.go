package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type responseCapture struct {
	http.ResponseWriter
	status int
}

func (w *responseCapture) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *responseCapture) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func productionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			var value [12]byte
			_, _ = rand.Read(value[:])
			requestID = hex.EncodeToString(value[:])
		}
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(self), microphone=(), geolocation=()")
		if !strings.Contains(r.URL.Path, "/media/posters") {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		}
		capture := &responseCapture{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("request panic", "request_id", requestID, "error", recovered)
				writeJSON(capture, http.StatusInternalServerError, map[string]string{"message": "Internal server error."})
			}
			slog.Info("http request", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "status", capture.status, "duration_ms", time.Since(started).Milliseconds())
		}()
		next.ServeHTTP(capture, r)
	})
}

type rateWindow struct {
	started time.Time
	count   int
}
type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]rateWindow
	maximum int
	window  time.Duration
}

func newRateLimiter(maximum int, window time.Duration) func(http.Handler) http.Handler {
	limiter := &rateLimiter{clients: make(map[string]rateWindow), maximum: maximum, window: window}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			now := time.Now()
			limiter.mu.Lock()
			entry := limiter.clients[ip]
			if entry.started.IsZero() || now.Sub(entry.started) >= limiter.window {
				entry = rateWindow{started: now}
			}
			entry.count++
			limiter.clients[ip] = entry
			limited := entry.count > limiter.maximum
			if len(limiter.clients) > 10_000 {
				for key, value := range limiter.clients {
					if now.Sub(value.started) >= limiter.window {
						delete(limiter.clients, key)
					}
				}
			}
			limiter.mu.Unlock()
			if limited {
				w.Header().Set("Retry-After", "60")
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"message": "Too many requests. Please try again shortly."})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
