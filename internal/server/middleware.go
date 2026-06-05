package server

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"code-review-agent/internal/storage"
)

type contextKey string

const (
	ctxUserID contextKey = "userID"
	ctxRole   contextKey = "role"
)

func ctxGetUserID(r *http.Request) int64 {
	v, _ := r.Context().Value(ctxUserID).(int64)
	return v
}

func ctxGetRole(r *http.Request) string {
	v, _ := r.Context().Value(ctxRole).(string)
	return v
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logger logs method, path, status code and duration for every request.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

// Recoverer catches panics, logs the stack trace and returns 500.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC: %v\n%s", rec, debug.Stack())
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// JSONContentType forces Content-Type: application/json on every response.
func JSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin rejects requests from non-admin users.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ctxGetRole(r) != "admin" {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"admin access required"}`)) //nolint:errcheck
				return
			}
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SessionAuth checks the session cookie (browser) or Bearer token (API).
// Unauthenticated browser requests are redirected to /login.
// Unauthenticated API requests receive 401 JSON.
func SessionAuth(store *storage.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""

			// 1. Cookie (browser)
			if c, err := r.Cookie("session"); err == nil {
				token = c.Value
			}

			// 2. Bearer token (API clients)
			if token == "" {
				if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
					token = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			if token != "" && store != nil {
				userID, err := store.GetSessionUserID(token)
				if err == nil && userID != 0 {
					role, _ := store.GetUserRole(userID)
					ctx := context.WithValue(r.Context(), ctxUserID, userID)
					ctx = context.WithValue(ctx, ctxRole, role)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// API requests get 401, browser requests get redirect to /login
			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"authentication required"}`)) //nolint:errcheck
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		})
	}
}
