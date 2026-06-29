package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/RealBlxckCodex/Aurora/internal/config"
	"github.com/RealBlxckCodex/Aurora/internal/inference"
)

type Server struct {
	cfg      *config.Config
	backends map[string]inference.IInferenceBackend
	httpSrv  *http.Server
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg:      cfg,
		backends: make(map[string]inference.IInferenceBackend),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/status", s.handleStatus)
	mux.HandleFunc("POST /v1/audio/speech", s.handleTTS)
	mux.HandleFunc("POST /v1/audio/transcriptions", s.handleSTT)
	mux.HandleFunc("GET /v1/models", s.handleListModels)
	mux.HandleFunc("GET /v1/languages", s.handleListLanguages)

	s.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: s.withCORS(s.withLogging(mux)),
	}

	return s
}

func (s *Server) RegisterBackend(id string, backend inference.IInferenceBackend) {
	s.backends[id] = backend
}

func (s *Server) Start() error {
	addr := s.httpSrv.Addr
	log.Printf("Aurora API server starting on %s", addr)
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.API.CORS.Enabled {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
