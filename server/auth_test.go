package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthHandlers(t *testing.T) {
	// Mock DB
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			if username == "admin" && password == "adminpass" {
				return "admin-id", true, nil
			}
			if username == "user" && password == "userpass" {
				return "user-id", false, nil
			}
			if username == "error" {
				return "", false, errors.New("db error")
			}
			return "", false, status.Error(codes.Unauthenticated, "invalid credentials") // Proper gRPC error
		},
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			if user.Username == "error" {
				return "", errors.New("db error")
			}
			return "new-user-id", nil
		},
		VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
			if password == "oldpassword" {
				return nil
			}
			if password == "error" {
				return errors.New("db error")
			}
			return status.Error(codes.Unauthenticated, "invalid password") // Proper gRPC error
		},
		UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
			if newPassword == "error" {
				return errors.New("db error")
			}
			return nil
		},
	}

	server := NewServer(mockDB, nil)

	// --- Login Tests ---

	t.Run("Login Success", func(t *testing.T) {
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
	})

	t.Run("Login - Bad Request (Missing Fields)", func(t *testing.T) {
		testCases := []LoginRequest{
			{Username: "", Password: "password"},
			{Username: "user", Password: ""},
		}

		for _, tc := range testCases {
			body, _ := json.Marshal(tc)
			req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for missing fields, got %d", w.Code)
			}
		}
	})

	t.Run("Login - Unauthorized (Invalid Credentials)", func(t *testing.T) {
		reqBody := LoginRequest{Username: "admin", Password: "wrongpass"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})

	t.Run("Login - Internal Server Error", func(t *testing.T) {
		reqBody := LoginRequest{Username: "error", Password: "any"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})

	// --- Create User Tests ---

	t.Run("Create User - Admin Success", func(t *testing.T) {
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

	t.Run("Create User - Bad Request (Missing Fields)", func(t *testing.T) {
		adminToken, _ := GenerateToken("admin-id", true)
		testCases := []CreateUserReq{
			{Username: "", Password: "p", DisplayName: "d"},
			{Username: "u", Password: "", DisplayName: "d"},
			{Username: "u", Password: "p", DisplayName: ""},
		}

		for _, tc := range testCases {
			body, _ := json.Marshal(tc)
			req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+adminToken)
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for missing fields, got %d", w.Code)
			}
		}
	})

	t.Run("Create User - Forbidden (Non-Admin)", func(t *testing.T) {
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

	t.Run("Create User - Internal Server Error", func(t *testing.T) {
		adminToken, _ := GenerateToken("admin-id", true)
		reqBody := CreateUserReq{Username: "error", Password: "p", DisplayName: "d"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})

	// --- Change Password Tests ---

	t.Run("Change Password - Success", func(t *testing.T) {
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

	t.Run("Change Password - Bad Request (Missing Fields)", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)
		testCases := []ChangePasswordReq{
			{OldPassword: "", NewPassword: "new"},
			{OldPassword: "old", NewPassword: ""},
		}

		for _, tc := range testCases {
			body, _ := json.Marshal(tc)
			req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+userToken)
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for missing fields, got %d", w.Code)
			}
		}
	})

	t.Run("Change Password - Unauthorized (Wrong Old Password)", func(t *testing.T) {
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

	t.Run("Change Password - Internal Server Error (Verify)", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)

		reqBody := ChangePasswordReq{OldPassword: "error", NewPassword: "newpassword"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})

	t.Run("Change Password - Internal Server Error (Update)", func(t *testing.T) {
		userToken, _ := GenerateToken("user-id", false)

		reqBody := ChangePasswordReq{OldPassword: "oldpassword", NewPassword: "error"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})

	t.Run("Change Password - Extra Spaces in Header", func(t *testing.T) {
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
