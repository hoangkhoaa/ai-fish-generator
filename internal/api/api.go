package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"fish-generate/internal/api/handlers"
	"fish-generate/internal/api/middleware"
	"fish-generate/internal/api/service"
	"fish-generate/internal/data"
	"fish-generate/internal/storage"
)

// Server represents the API server
type Server struct {
	server      *http.Server
	router      *mux.Router
	storage     storage.StorageAdapter
	dataManager *data.DataManager
}

// Config holds the API server configuration
type Config struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	Storage      storage.StorageAdapter
	DataManager  *data.DataManager
}

// DefaultConfig returns the default server configuration
func DefaultConfig() Config {
	return Config{
		Port:         "8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// NewServer creates a new API server
func NewServer(cfg Config) *Server {
	router := mux.NewRouter()

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &Server{
		server:      server,
		router:      router,
		storage:     cfg.Storage,
		dataManager: cfg.DataManager,
	}
}

// Start initializes and starts the API server
func (s *Server) Start() error {
	// Initialize the fishing service
	fishingService := service.NewFishingService(s.storage, s.dataManager)

	// Initialize the fishing handler
	fishingHandler := handlers.NewFishingHandler(fishingService)

	// Set up API routes
	apiRouter := s.router.PathPrefix("/api").Subrouter()

	// Health check endpoint
	s.router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Apply middleware to all routes
	fishCatchHandler := middleware.ApplyMiddleware(
		fishingHandler.CatchFish,
		middleware.Logging(),
		middleware.CORS(),
	)

	regionsHandler := middleware.ApplyMiddleware(
		fishingHandler.GetRegions,
		middleware.Logging(),
		middleware.CORS(),
	)

	conditionsHandler := middleware.ApplyMiddleware(
		fishingHandler.GetCurrentConditions,
		middleware.Logging(),
		middleware.CORS(),
	)

	// Register routes
	apiRouter.HandleFunc("/fish", fishCatchHandler).Methods(http.MethodGet, http.MethodOptions)
	apiRouter.HandleFunc("/regions", regionsHandler).Methods(http.MethodGet, http.MethodOptions)
	apiRouter.HandleFunc("/conditions", conditionsHandler).Methods(http.MethodGet, http.MethodOptions)

	log.Printf("API server starting on port %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the API server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("API server shutting down...")
	return s.server.Shutdown(ctx)
}
