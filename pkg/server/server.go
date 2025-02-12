package server

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"pihole-linktree/pkg/cache"
	"pihole-linktree/pkg/config"
	"pihole-linktree/pkg/health"
)

type Server struct {
	cache  *cache.Cache
	server *http.Server
}

func New(cache *cache.Cache, cfg *config.Config) *Server {
	mux := http.NewServeMux()

	s := &Server{
		cache: cache,
		server: &http.Server{
			Addr:    ":8080",
			Handler: mux,
		},
	}

	// Register routes
	mux.HandleFunc("GET /", s.handleHomePage())
	mux.HandleFunc("GET /health", health.Handler(cache, cfg))

	return s
}

func (s *Server) handleHomePage() http.HandlerFunc {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		info := s.cache.Get()
		if info == nil {
			http.Error(w, "Data not yet available", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.Execute(w, info); err != nil {
			log.Printf("Error rendering template: %v", err)
			return
		}
	}
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
