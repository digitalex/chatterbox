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
	"google.golang.org/api/iterator"
)

// Server holds dependencies
type Server struct {
	spannerClient *spanner.Client
	router        *chi.Mux
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

	// Initialize Server
	srv := NewServer(client)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s", port)
	http.ListenAndServe(":"+port, srv.router)
}

func (s *Server) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	stmt := spanner.Statement{SQL: "SELECT 1"}
	iter := s.spannerClient.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		http.Error(w, "No results", http.StatusInternalServerError)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Spanner Error: %v", err), http.StatusInternalServerError)
		return
	}

	var val int64
	if err := row.Column(0, &val); err != nil {
		http.Error(w, "Parse Error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"db_check": val,
	})
}

// NewServer sets up routes and returns the server struct
func NewServer(client *spanner.Client) *Server {
	s := &Server{
		spannerClient: client,
		router:        chi.NewRouter(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-User-ID"},
	}))

	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Chatterbox API is running 🚀"))
	})

	s.router.Get("/health", s.healthCheckHandler)
	s.router.Post("/api/sync", s.syncHandler)
	s.router.Post("/api/rooms/{roomID}/messages", s.sendMessageHandler)
}

