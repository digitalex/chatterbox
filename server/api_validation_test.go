package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestLoginValidation tests valid and invalid login scenarios
func TestLoginValidation(t *testing.T) {
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			if username == "valid" && password == "valid" {
				return "user-id", false, nil
			}
			if username == "error" {
				return "", false, errors.New("db error")
			}
			// Invalid credentials return empty ID and nil error (as per memory/interface contract)
			return "", false, nil
		},
	}
	server := NewServer(mockDB, nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "Valid Login",
			body:       `{"username": "valid", "password": "valid"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid Credentials",
			body:       `{"username": "invalid", "password": "wrong"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Missing Password",
			body:       `{"username": "valid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing Username",
			body:       `{"password": "valid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "DB Error",
			body:       `{"username": "error", "password": "any"}`,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/login", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Test %s: Expected status %d, got %d. Body: %s", tt.name, tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestCreateUserValidation tests input validation for creating users
func TestCreateUserValidation(t *testing.T) {
	mockDB := &MockDB{
		CreateUserFn: func(ctx context.Context, req CreateUserReq) (string, error) {
			return "new-id", nil
		},
	}
	server := NewServer(mockDB, nil)
	adminToken, _ := GenerateToken("admin", true)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "Valid User",
			body:       `{"username": "new", "password": "pw", "display_name": "New User"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Missing Display Name",
			body:       `{"username": "new", "password": "pw"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing Username",
			body:       `{"password": "pw", "display_name": "New User"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing Password",
			body:       `{"username": "new", "display_name": "New User"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/users", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+adminToken)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Test %s: Expected status %d, got %d", tt.name, tt.wantStatus, w.Code)
			}
		})
	}
}

// TestChangePasswordValidation tests validation for changing passwords
func TestChangePasswordValidation(t *testing.T) {
	mockDB := &MockDB{
		VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
			if password == "old" {
				return nil
			}
			return errors.New("invalid")
		},
		UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
			return nil
		},
	}
	server := NewServer(mockDB, nil)
	token, _ := GenerateToken("user", false)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "Valid Change",
			body:       `{"old_password": "old", "new_password": "new"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Missing Old Password",
			body:       `{"new_password": "new"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing New Password",
			body:       `{"old_password": "old"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/change-password", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Test %s: Expected status %d, got %d", tt.name, tt.wantStatus, w.Code)
			}
		})
	}
}

// TestSyncValidation tests input validation for sync
func TestSyncValidation(t *testing.T) {
	mockDB := &MockDB{
		SyncFn: func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
			return []*RoomResult{}, []*MsgResult{}, []*UserResult{}, nil
		},
	}
	server := NewServer(mockDB, nil)
	token, _ := GenerateToken("user", false)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "Valid Sync",
			body:       `{"rooms": [], "messages": []}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid JSON",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid Room (Missing Name)",
			body:       `{"rooms": [{"room_id": "r1"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid Message (Missing RoomID)",
			body:       `{"messages": [{"message_id": 1}]}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Test %s: Expected status %d, got %d", tt.name, tt.wantStatus, w.Code)
			}
		})
	}
}
