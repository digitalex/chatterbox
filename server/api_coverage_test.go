package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- Login Tests ---

func TestLoginBadRequest(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)

	// Missing password
	reqBody := `{"username": "user"}`
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(reqBody))
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}

	// Missing username
	reqBody = `{"password": "pass"}`
	req = httptest.NewRequest("POST", "/api/login", strings.NewReader(reqBody))
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}
}

func TestLoginUnauthorized(t *testing.T) {
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			return "", false, status.Error(codes.Unauthenticated, "invalid credentials")
		},
	}
	srv := NewServer(mockDB, nil)

	reqBody := `{"username": "user", "password": "wrongpassword"}`
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(reqBody))
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestLoginError(t *testing.T) {
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			return "", false, fmt.Errorf("db error")
		},
	}
	srv := NewServer(mockDB, nil)

	reqBody := `{"username": "user", "password": "password"}`
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(reqBody))
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 Internal Server Error, got %d", w.Code)
	}
}

// --- Create User Tests ---

func TestCreateUserBadRequest(t *testing.T) {
	adminToken, _ := GenerateToken("admin-id", true)
	srv := NewServer(&MockDB{}, nil)

	// Missing password
	reqBody := `{"username": "newuser", "display_name": "User"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}
}

func TestCreateUserUnauthorizedMissingToken(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)
	reqBody := `{"username": "newuser", "password": "pw", "display_name": "User"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(reqBody))
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestCreateUserError(t *testing.T) {
	adminToken, _ := GenerateToken("admin-id", true)
	mockDB := &MockDB{
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "", fmt.Errorf("db error")
		},
	}
	srv := NewServer(mockDB, nil)

	reqBody := `{"username": "newuser", "password": "pw", "display_name": "User"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 Internal Server Error, got %d", w.Code)
	}
}

// --- Change Password Tests ---

func TestChangePasswordBadRequest(t *testing.T) {
	userToken, _ := GenerateToken("user-id", false)
	srv := NewServer(&MockDB{}, nil)

	// Missing new_password
	reqBody := `{"old_password": "old"}`
	req := httptest.NewRequest("POST", "/api/change-password", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}
}

func TestChangePasswordUnauthorizedMissingToken(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)
	reqBody := `{"old_password": "old", "new_password": "new"}`
	req := httptest.NewRequest("POST", "/api/change-password", strings.NewReader(reqBody))
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestChangePasswordError(t *testing.T) {
	userToken, _ := GenerateToken("user-id", false)
	mockDB := &MockDB{
		VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
			return nil
		},
		UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
			return fmt.Errorf("db error")
		},
	}
	srv := NewServer(mockDB, nil)

	reqBody := `{"old_password": "old", "new_password": "new"}`
	req := httptest.NewRequest("POST", "/api/change-password", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 Internal Server Error, got %d", w.Code)
	}
}

// --- Sync Tests ---

func TestSyncBadRequest(t *testing.T) {
	userToken, _ := GenerateToken("user-id", false)
	srv := NewServer(&MockDB{}, nil)

	// Invalid JSON
	reqBody := `{"last_synced_at": `
	req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for invalid JSON, got %d", w.Code)
	}

	// Missing room name
	reqBody = `{"rooms": [{"room_id": "r1"}]}` // Missing name
	req = httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing room name, got %d", w.Code)
	}

	// Missing room id
	reqBody = `{"rooms": [{"name": "r1"}]}` // Missing room_id
	req = httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing room id, got %d", w.Code)
	}

	// Missing message room_id
	reqBody = `{"messages": [{"message_id": 1}]}` // Missing room_id
	req = httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing message room_id, got %d", w.Code)
	}
}

// --- Get Room Members Tests ---

func TestGetRoomMembersUnauthorized(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)
	req := httptest.NewRequest("GET", "/api/rooms/r1/members", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}
}

// --- Update Profile Tests ---
func TestUpdateProfileUnauthorized(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)
	req := httptest.NewRequest("POST", "/api/me", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}
}
