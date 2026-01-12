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
			return "", false, nil // Error?
		},
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "new-user-id", nil
		},
		VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
			if password == "oldpassword" {
				return nil
			}
			return http.ErrNoCookie // Just an error
		},
		UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
			return nil
		},
	}

	server := NewServer(mockDB, nil)

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
		// Verify strict JSON structure if possible, but map check is usually better for extra fields.
		// For now, ensuring Token exists is good.
	})

	t.Run("Create User - Admin", func(t *testing.T) {
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

	t.Run("Create User - NonAdmin", func(t *testing.T) {
		// Generate User Token
		userToken, _ := GenerateToken("user-id", false)

		reqBody := CreateUserReq{Username: "newuser", Password: "password"}
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

	t.Run("Change Password - Wrong Old Password", func(t *testing.T) {
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

func TestAuthValidation(t *testing.T) {
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			if username == "valid" && password == "valid" {
				return "user-id", false, nil
			}
			if username == "valid" && password == "wrong" {
				return "", false, status.Error(codes.Unauthenticated, "invalid credentials")
			}
			return "", false, fmt.Errorf("db error")
		},
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "new-id", nil
		},
		UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
			return nil
		},
	}
	server := NewServer(mockDB, nil)

	// --- Login Validation ---
	t.Run("Login 400 - Missing Username", func(t *testing.T) {
		reqBody := LoginRequest{Password: "pass"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("Login 400 - Missing Password", func(t *testing.T) {
		reqBody := LoginRequest{Username: "user"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("Login 401 - Invalid Credentials", func(t *testing.T) {
		reqBody := LoginRequest{Username: "valid", Password: "wrong"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})

	t.Run("Login 500 - Internal Error", func(t *testing.T) {
		reqBody := LoginRequest{Username: "other", Password: "other"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})

	// --- Create User Validation ---
	t.Run("Create User 400 - Missing Fields", func(t *testing.T) {
		adminToken, _ := GenerateToken("admin", true)
		cases := []CreateUserReq{
			{Password: "p", DisplayName: "d"},             // Missing Username
			{Username: "u", DisplayName: "d"},             // Missing Password
			{Username: "u", Password: "p"},                // Missing DisplayName
		}

		for _, c := range cases {
			body, _ := json.Marshal(c)
			req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+adminToken)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for missing fields, got %d", w.Code)
			}
		}
	})

	t.Run("Create User 500 - DB Error", func(t *testing.T) {
		mockDB.CreateUserFn = func(ctx context.Context, user CreateUserReq) (string, error) {
			return "", fmt.Errorf("db error")
		}
		adminToken, _ := GenerateToken("admin", true)
		reqBody := CreateUserReq{Username: "u", Password: "p", DisplayName: "d"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})

	// --- Change Password Validation ---
	t.Run("Change Password 400 - Missing Fields", func(t *testing.T) {
		userToken, _ := GenerateToken("user", false)
		cases := []ChangePasswordReq{
			{NewPassword: "new"}, // Missing Old
			{OldPassword: "old"}, // Missing New
		}

		for _, c := range cases {
			body, _ := json.Marshal(c)
			req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+userToken)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for missing fields, got %d", w.Code)
			}
		}
	})

	t.Run("Change Password 500 - DB Error", func(t *testing.T) {
		// Mock UpdatePassword to fail
		mockDB.UpdatePasswordFn = func(ctx context.Context, userID, newPassword string) error {
			return fmt.Errorf("db error")
		}
		// Mock VerifyPassword to succeed so we reach UpdatePassword
		mockDB.VerifyPasswordFn = func(ctx context.Context, userID, password string) error {
			return nil
		}

		userToken, _ := GenerateToken("user", false)
		reqBody := ChangePasswordReq{OldPassword: "old", NewPassword: "new"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}
