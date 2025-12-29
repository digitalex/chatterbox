package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthCoverage(t *testing.T) {
	// Mock DB
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			if username == "admin" && password == "adminpass" {
				return "admin-id", true, nil
			}
			if username == "error" {
				return "", false, fmt.Errorf("DB Error")
			}
			// Simulate invalid credentials return
			return "", false, status.Error(codes.Unauthenticated, "invalid credentials")
		},
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			if user.Username == "error" {
				return "", fmt.Errorf("DB Error")
			}
			return "new-user-id", nil
		},
		VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
			if password == "oldpassword" {
				return nil
			}
			return status.Error(codes.Unauthenticated, "invalid password")
		},
		UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
			if newPassword == "error" {
				return fmt.Errorf("DB Error")
			}
			return nil
		},
	}

	server := NewServer(mockDB, nil)

	// POST /api/login Coverage
	t.Run("Login - Missing Fields", func(t *testing.T) {
		reqBody := LoginRequest{Username: ""} // Missing password
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing fields, got %d", w.Code)
		}
	})

	t.Run("Login - Invalid Credentials", func(t *testing.T) {
		reqBody := LoginRequest{Username: "wrong", Password: "wrong"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 for invalid credentials, got %d", w.Code)
		}
	})

	t.Run("Login - DB Error", func(t *testing.T) {
		reqBody := LoginRequest{Username: "error", Password: "any"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 for DB error, got %d", w.Code)
		}
	})

	// POST /api/users Coverage
	t.Run("Create User - Missing Fields", func(t *testing.T) {
		adminToken, _ := GenerateToken("admin-id", true)
		reqBody := CreateUserReq{Username: "user"} // Missing password and display_name
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing fields, got %d", w.Code)
		}
	})

	t.Run("Create User - DB Error", func(t *testing.T) {
		adminToken, _ := GenerateToken("admin-id", true)
		reqBody := CreateUserReq{Username: "error", Password: "p", DisplayName: "d"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 for DB error, got %d", w.Code)
		}
	})

	// POST /api/change-password Coverage
	t.Run("Change Password - Missing Fields", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)
		reqBody := ChangePasswordReq{OldPassword: ""} // Missing old password
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing fields, got %d", w.Code)
		}
	})

	t.Run("Change Password - Invalid Old Password", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)
		reqBody := ChangePasswordReq{OldPassword: "wrongpassword", NewPassword: "newpassword"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 for invalid old password, got %d", w.Code)
		}
	})

	t.Run("Change Password - DB Error", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)
		reqBody := ChangePasswordReq{OldPassword: "oldpassword", NewPassword: "error"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 for DB error, got %d", w.Code)
		}
	})

	// POST /api/sync Coverage
	t.Run("Sync - Invalid JSON", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)
		reqBody := `{"rooms": "invalid"}` // Invalid JSON structure for rooms
		req := httptest.NewRequest("POST", "/api/sync", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("Sync - Missing Fields", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)
		reqBody := `{"rooms": [{"room_id": "", "name": "room"}]}` // Missing room_id
		req := httptest.NewRequest("POST", "/api/sync", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing fields, got %d", w.Code)
		}
	})
}
