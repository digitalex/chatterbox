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

var spannerClient *spanner.Client

func main() {
	ctx := context.Background()
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT") // e.g. "alexander-chatterbox"
	instanceID := "chatterbox-db"
	databaseID := os.Getenv("SPANNER_DATABASE") // e.g. "chatter-test"

	if projectID == "" || databaseID == "" {
		log.Fatal("Please set GOOGLE_CLOUD_PROJECT and SPANNER_DATABASE env vars")
	}

	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)

	// 1. Connect to Spanner
	var err error
	spannerClient, err = spanner.NewClient(ctx, dbPath)
	if err != nil {
		log.Fatalf("Failed to create Spanner client: %v", err)
	}
	defer spannerClient.Close()
	log.Println("✅ Connected to Spanner at", dbPath)

	// 2. Setup Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	
	// Basic CORS for development
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"}, // Change to your Firebase URL in prod
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-User-ID"},
	}))

	// 3. Define Routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Chatterbox API is running 🚀"))
	})

	r.Get("/health", healthCheckHandler)

	// Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s", port)
	http.ListenAndServe(":"+port, r)
}

// healthCheckHandler runs a simple query to ensure DB access works
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	stmt := spanner.Statement{SQL: "SELECT 1"}
	iter := spannerClient.Single().Query(r.Context(), stmt)
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

