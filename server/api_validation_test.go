package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAPIGaps(t *testing.T) {
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			if username == "valid" && password == "valid" {
				return "user-id", false, nil
			}
			return "", false, fmt.Errorf("invalid credentials")
		},
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "new-user-id", nil
		},
	}
	server := NewServer(mockDB, nil)

	t.Run("Login - Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/login", strings.NewReader(`{"username":`)) // Broken JSON
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("Login - Missing Fields", func(t *testing.T) {
		// Missing password
		req, _ := http.NewRequest("POST", "/api/login", strings.NewReader(`{"username": "user"}`))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing password, got %d", w.Code)
		}
	})

	t.Run("Login - Invalid Credentials", func(t *testing.T) {
		reqBody := LoginRequest{Username: "wrong", Password: "wrong"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 for invalid credentials, got %d", w.Code)
		}
	})

	t.Run("CreateUser - Missing DisplayName", func(t *testing.T) {
		adminToken, _ := GenerateToken("admin-id", true)
		reqBody := CreateUserReq{Username: "newuser", Password: "password"} // Missing DisplayName
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// This is expected to fail currently, as code doesn't check DisplayName
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing DisplayName, got %d", w.Code)
		}
	})

	t.Run("ChangePassword - Invalid JSON", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)
		req, _ := http.NewRequest("POST", "/api/change-password", strings.NewReader(`{`)) // Broken JSON
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("ChangePassword - Missing Fields", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)
		reqBody := ChangePasswordReq{NewPassword: "new"} // Missing OldPassword
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing fields, got %d", w.Code)
		}
	})

	t.Run("Sync - Invalid JSON", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)
		req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(`{"last_synced_at":`)) // Broken JSON
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// This is expected to fail currently, as code ignores decode errors
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON in sync, got %d", w.Code)
		}
	})

	t.Run("Sync - Empty Body", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)
		req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(``)) // Empty body
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Per OpenAPI, body is required.
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty body in sync, got %d", w.Code)
		}
	})
}
