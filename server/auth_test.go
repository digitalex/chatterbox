package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthHandlers(t *testing.T) {
	// Helper to reset mock DB for each test
	setup := func() *MockDB {
		return &MockDB{
			AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
				if username == "admin" && password == "adminpass" {
					return "admin-id", true, nil
				}
				if username == "user" && password == "userpass" {
					return "user-id", false, nil
				}
				// Default failure
				return "", false, status.Error(codes.Unauthenticated, "Invalid credentials")
			},
			CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
				return "new-user-id", nil
			},
			VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
				if password == "oldpassword" {
					return nil
				}
				return status.Error(codes.Unauthenticated, "Invalid old password")
			},
			UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
				return nil
			},
		}
	}

	t.Run("Login Success", func(t *testing.T) {
		mockDB := setup()
		server := NewServer(mockDB, nil)

		reqBody := LoginRequest{Username: "admin", Password: "adminpass"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp LoginResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if resp.Token == "" {
			t.Error("Expected token")
		}
		// Verify strict JSON structure if possible, but map check is usually better for extra fields.
		// For now, ensuring Token exists is good.
	})

	t.Run("Login - Bad Request", func(t *testing.T) {
		mockDB := setup()
		server := NewServer(mockDB, nil)

		reqBody := LoginRequest{Username: "", Password: ""}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("Login - Internal Error", func(t *testing.T) {
		mockDB := setup()
		// Mock DB to return error for specific user
		mockDB.AuthenticateUserFn = func(ctx context.Context, username, password string) (string, bool, error) {
			// Simulate DB error that is NOT unauthenticated
			return "", false, context.DeadlineExceeded // Example generic error
		}
		server := NewServer(mockDB, nil)

		reqBody := LoginRequest{Username: "db-error", Password: "password"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})

	t.Run("Create User - Admin", func(t *testing.T) {
		mockDB := setup()
		server := NewServer(mockDB, nil)

		// Generate Admin Token
		adminToken, _ := GenerateToken("admin-id", true)

		reqBody := CreateUserReq{Username: "newuser", Password: "password", DisplayName: "New User"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if resp["user_id"] != "new-user-id" {
			t.Errorf("Expected user_id 'new-user-id', got %v", resp["user_id"])
		}
	})

	t.Run("Create User - Bad Request", func(t *testing.T) {
		mockDB := setup()
		server := NewServer(mockDB, nil)
		adminToken, _ := GenerateToken("admin-id", true)

		// Missing display_name, which is required
		reqBody := CreateUserReq{Username: "newuser", Password: "password"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("Create User - Internal Error", func(t *testing.T) {
		mockDB := setup()
		mockDB.CreateUserFn = func(ctx context.Context, user CreateUserReq) (string, error) {
			return "", context.DeadlineExceeded
		}
		server := NewServer(mockDB, nil)
		adminToken, _ := GenerateToken("admin-id", true)

		reqBody := CreateUserReq{Username: "newuser", Password: "password", DisplayName: "New User"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})

	t.Run("Create User - NonAdmin", func(t *testing.T) {
		mockDB := setup()
		server := NewServer(mockDB, nil)

		// Generate User Token
		userToken, _ := GenerateToken("user-id", false)

		reqBody := CreateUserReq{Username: "newuser", Password: "password", DisplayName: "New User"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
	})

	t.Run("Change Password", func(t *testing.T) {
		mockDB := setup()
		server := NewServer(mockDB, nil)
		userToken, _ := GenerateToken("user-id", false)

		reqBody := ChangePasswordReq{OldPassword: "oldpassword", NewPassword: "newpassword"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if resp["status"] != "ok" {
			t.Errorf("Expected status 'ok', got %v", resp["status"])
		}
	})

	t.Run("Change Password - Bad Request", func(t *testing.T) {
		mockDB := setup()
		server := NewServer(mockDB, nil)
		userToken, _ := GenerateToken("user-id", false)

		// Missing new password
		reqBody := ChangePasswordReq{OldPassword: "oldpassword"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("Change Password - Internal Error", func(t *testing.T) {
		mockDB := setup()
		mockDB.UpdatePasswordFn = func(ctx context.Context, userID, newPassword string) error {
			return context.DeadlineExceeded
		}
		server := NewServer(mockDB, nil)
		userToken, _ := GenerateToken("user-id", false)

		reqBody := ChangePasswordReq{OldPassword: "oldpassword", NewPassword: "newpassword"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})

	t.Run("Change Password - Wrong Old Password", func(t *testing.T) {
		mockDB := setup()
		server := NewServer(mockDB, nil)
		userToken, _ := GenerateToken("user-id", false)

		reqBody := ChangePasswordReq{OldPassword: "wrongpassword", NewPassword: "newpassword"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})

	t.Run("Change Password - Extra Spaces in Header", func(t *testing.T) {
		mockDB := setup()
		server := NewServer(mockDB, nil)
		userToken, _ := GenerateToken("user-id", false)

		reqBody := ChangePasswordReq{OldPassword: "oldpassword", NewPassword: "newpassword"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		// Note double space
		req.Header.Set("Authorization", "Bearer  "+userToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d, body: %s", w.Code, w.Body.String())
		}
	})
}
