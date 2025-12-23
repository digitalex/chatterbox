package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIValidation(t *testing.T) {
	mockDB := &MockDB{}
	server := NewServer(mockDB, nil)

	// Helper to generate a token
	token, _ := GenerateToken("user-id", false)

	t.Run("Sync - Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sync", bytes.NewBuffer([]byte(`{invalid-json}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Expect 400 Bad Request
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

    t.Run("Sync - Empty Body", func(t *testing.T) {
        // Spec says body is required
        req := httptest.NewRequest("POST", "/api/sync", bytes.NewBuffer(nil))
        req.Header.Set("Authorization", "Bearer "+token)
        w := httptest.NewRecorder()

        server.router.ServeHTTP(w, req)

        // Note: Currently code might handle this as "empty sync", but strict API validation might say 400.
        // If code swallows it, this test will fail if I assert 400.
        // Let's check what the spec says: "requestBody: required: true".
        // Go's json decoder on empty body returns EOF, which is an error.
        // So if we fix the code to return error on any Decode error, this should be 400.
        if w.Code != http.StatusBadRequest {
             t.Errorf("Expected 400 for empty body, got %d", w.Code)
        }
    })

	t.Run("Update Profile - Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/me", bytes.NewBuffer([]byte(`{invalid}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

    t.Run("Update Profile - Missing Fields (Allowed)", func(t *testing.T) {
        // "required" is not set for fields in ProfileReq, so empty object is valid JSON but does nothing?
        req := httptest.NewRequest("POST", "/api/me", bytes.NewBuffer([]byte(`{}`)))
        req.Header.Set("Authorization", "Bearer "+token)
        w := httptest.NewRecorder()

        server.router.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
            t.Errorf("Expected 200, got %d", w.Code)
        }
    })

	t.Run("Login - Missing Fields", func(t *testing.T) {
		reqBody := LoginRequest{Username: "", Password: ""}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

    t.Run("Login - Invalid JSON", func(t *testing.T) {
        req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer([]byte(`bad`)))
        w := httptest.NewRecorder()

        server.router.ServeHTTP(w, req)

        if w.Code != http.StatusBadRequest {
            t.Errorf("Expected 400, got %d", w.Code)
        }
    })
}
