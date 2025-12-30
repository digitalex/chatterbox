package main

import (
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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
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
		r.Post("/api/me", s.updateProfileHandler)
		r.Get("/api/rooms/{roomID}/members", s.getRoomMembersHandler)
		r.Post("/api/users", s.createUserHandler)
		r.Post("/api/change-password", s.changePasswordHandler)
	})
}

func (s *Server) cloudLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap body to capture it as it is read
		var capturer *bodyCapturer
		if r.Body != nil {
			capturer = newBodyCapturer(r.Body, 1024*1024) // 1MB limit
			r.Body = capturer
		}

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		if ww.Status() >= 400 {
			var bodyBytes []byte
			if capturer != nil {
				// We need to ensure we have the body. If the handler didn't read it,
				// we read it now. If the handler read it partially, we read the rest.
				// We only care about capturing up to the limit (1MB).
				// We do NOT want to drain the entire body if it exceeds the limit, as that could cause DoS.
				remainingToCapture := 1024*1024 - int64(capturer.buf.Len())
				if remainingToCapture > 0 {
					io.CopyN(io.Discard, capturer, remainingToCapture)
				}
				bodyBytes = capturer.GetBody()
			}

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
