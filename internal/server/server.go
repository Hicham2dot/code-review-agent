package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"code-review-agent/internal/config"
	"code-review-agent/internal/storage"
)

// Server holds the HTTP server and its dependencies.
type Server struct {
	cfg     config.Config
	store   *storage.Store
	router  chi.Router
	httpSrv *http.Server
	startAt time.Time
}

// New creates a Server wired with all routes and middleware.
func New(cfg config.Config, store *storage.Store) *Server {
	s := &Server{
		cfg:     cfg,
		store:   store,
		startAt: time.Now(),
	}
	s.router = s.buildRouter()
	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

func (s *Server) buildRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", s.healthHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/status", s.statusHandler)
		r.Post("/analyze", s.analyzeHandler)
		r.Get("/analyses", s.listAnalysesHandler)
		r.Get("/analyses/{hash}", s.getAnalysisHandler)
		r.Delete("/cache", s.clearCacheHandler)
	})

	r.Post("/webhook", s.webhookHandler)

	return r
}

// Start begins listening. Blocks until the server stops.
func (s *Server) Start() error {
	fmt.Printf("Server listening on %s\n", s.httpSrv.Addr)
	if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server within the given context deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

