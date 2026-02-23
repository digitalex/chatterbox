package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthHandlersEdgeCases(t *testing.T) {
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			if username == "validuser" && password == "validpass" {
				return "user1", false, nil
			}
			// Simulate invalid credentials by returning error
			return "", false, fmt.Errorf("invalid credentials")
		},
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "new-user-id", nil
		},
		VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
			if password == "correct-old-pass" {
				return nil
			}
			return fmt.Errorf("wrong password")
		},
		UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
			return nil
		},
	}
	server := NewServer(mockDB, nil)

	t.Run("Login Bad Request - Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBufferString(`{"username":`)) // Invalid JSON
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Login Bad Request - Missing Fields", func(t *testing.T) {
		reqBody := LoginRequest{Username: ""}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Login Failure - 401", func(t *testing.T) {
		reqBody := LoginRequest{Username: "validuser", Password: "wrongpassword"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Create User Bad Request - Invalid JSON", func(t *testing.T) {
		token, _ := GenerateToken("admin", true)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBufferString(`{`)) // Invalid JSON
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Create User Bad Request - Missing Fields", func(t *testing.T) {
		token, _ := GenerateToken("admin", true)
		reqBody := CreateUserReq{Username: "newuser"} // Missing Password
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Change Password Bad Request - Invalid JSON", func(t *testing.T) {
		token, _ := GenerateToken("user1", false)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBufferString(`{`)) // Invalid JSON
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Change Password Bad Request - Missing New Password", func(t *testing.T) {
		token, _ := GenerateToken("user1", false)
		reqBody := ChangePasswordReq{OldPassword: "correct-old-pass"} // Missing NewPassword
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Change Password Bad Request - Missing Old Password", func(t *testing.T) {
		token, _ := GenerateToken("user1", false)
		reqBody := ChangePasswordReq{NewPassword: "new-pass"} // Missing OldPassword
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})
}
