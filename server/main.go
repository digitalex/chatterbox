package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server holds dependencies
type Server struct {
	db     Database
	router *chi.Mux
}

// Sets up everything
func main() {
	ctx := context.Background()
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	instanceID := "chatterbox-db"
	databaseID := os.Getenv("SPANNER_DATABASE")

	if projectID == "" || databaseID == "" {
		log.Fatal("Please set GOOGLE_CLOUD_PROJECT and SPANNER_DATABASE")
	}

	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)

	client, err := spanner.NewClient(ctx, dbPath)
	if err != nil {
		log.Fatalf("Failed to create Spanner client: %v", err)
	}
	defer client.Close()

	// Initialize Server with SpannerDB
	srv := NewServer(&SpannerDB{client: client})
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s", port)
	http.ListenAndServe(":"+port, srv.router)
}

func (s *Server) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	val, err := s.db.HealthCheck(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Spanner Error: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"db_check": val,
	})
}

// NewServer sets up routes and returns the server struct
func NewServer(db Database) *Server {
	s := &Server{
		db:     db,
		router: chi.NewRouter(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:5173",             // Local Development
			"https://chatterbox-480916.web.app", // Your Production Frontend
			"https://chatterbox-480916.firebaseapp.com",
		},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-User-ID"},
		AllowCredentials: true,
	}))

	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Chatterbox API is running 🚀"))
	})

	s.router.Get("/health", s.healthCheckHandler)
	s.router.Post("/api/login", s.loginHandler)

	s.router.Group(func(r chi.Router) {
		r.Use(JWTMiddleware)
		r.Post("/api/sync", s.syncHandler)
		r.Post("/api/rooms", s.createRoomHandler)
		r.Post("/api/rooms/{roomID}/messages", s.sendMessageHandler)
		r.Post("/api/me", s.updateProfileHandler)
		r.Get("/api/rooms/{roomID}/members", s.getRoomMembersHandler)
	})
}
