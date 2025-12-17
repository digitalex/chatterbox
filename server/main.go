package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server holds dependencies
type Server struct {
	db     Database
	router *chi.Mux
	logger *logging.Client
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

	// Create logging client
	logClient, err := logging.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create logging client: %v", err)
	}
	defer logClient.Close()

	// Initialize Server with SpannerDB
	srv := NewServer(&SpannerDB{client: client}, logClient)

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
func NewServer(db Database, logger *logging.Client) *Server {
	s := &Server{
		db:     db,
		router: chi.NewRouter(),
		logger: logger,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(s.cloudLoggingMiddleware)
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
        r.Post("/api/rooms/{roomID}/invites", s.generateInviteHandler)
        r.Post("/api/invites/{inviteCode}/accept", s.acceptInviteHandler)
	})
}

func (s *Server) cloudLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body
		var bodyBytes []byte
		if r.Body != nil {
			// Limit request body read to 1MB to prevent DoS
			bodyBytes, _ = io.ReadAll(io.LimitReader(r.Body, 1024*1024))
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		if ww.Status() >= 400 {
			// Log to Cloud Logging
			if s.logger != nil {
				// We log asynchronously to avoid blocking the response?
				// But client library handles batching.
				s.logger.Logger("error-log").Log(logging.Entry{
					Payload: map[string]interface{}{
						"method":  r.Method,
						"url":     r.URL.String(),
						"headers": r.Header,
						"body":    string(bodyBytes),
						"status":  ww.Status(),
					},
					Severity: logging.Error,
				})
			}
		}
	})
}
