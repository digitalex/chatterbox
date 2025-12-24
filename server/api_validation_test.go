package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidation_Login(t *testing.T) {
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			if username == "valid" && password == "valid" {
				return "uid", false, nil
			}
			return "", false, nil // Invalid credentials
		},
	}
	server := NewServer(mockDB, nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "Missing Username",
			body:       `{"password": "valid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing Password",
			body:       `{"username": "valid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Empty Body",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid JSON",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Wrong Credentials",
			body:       `{"username": "valid", "password": "wrong"}`,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/login", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestValidation_CreateUser(t *testing.T) {
	mockDB := &MockDB{
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "uid", nil
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
			name:       "Missing Username",
			body:       `{"password": "p", "display_name": "d"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing Password",
			body:       `{"username": "u", "display_name": "d"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing DisplayName",
			body:       `{"username": "u", "password": "p"}`,
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
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestValidation_ChangePassword(t *testing.T) {
	mockDB := &MockDB{}
	server := NewServer(mockDB, nil)
	userToken, _ := GenerateToken("user", false)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
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
			req.Header.Set("Authorization", "Bearer "+userToken)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestValidation_Sync(t *testing.T) {
	mockDB := &MockDB{
        SyncFn: func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
			return []*RoomResult{}, []*MsgResult{}, []*UserResult{}, nil
		},
    }
	server := NewServer(mockDB, nil)
	userToken, _ := GenerateToken("user", false)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "Missing Room ID in Rooms",
			body:       `{"rooms": [{"name": "n"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing Name in Rooms",
			body:       `{"rooms": [{"room_id": "r"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing Room ID in Messages",
			body:       `{"messages": [{"message_id": 1, "content": "c"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing Message ID in Messages",
			body:       `{"messages": [{"room_id": "r", "content": "c"}]}`,
			wantStatus: http.StatusBadRequest,
		},
        {
            name:       "Valid Sync",
            body:       `{"rooms": [{"room_id": "r", "name": "n"}], "messages": [{"room_id": "r", "message_id": 1, "content": "c"}]}`,
            wantStatus: http.StatusOK,
        },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+userToken)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestErrors_Login(t *testing.T) {
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			return "", false, fmt.Errorf("db error")
		},
	}
	server := NewServer(mockDB, nil)

	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"username":"u", "password":"p"}`))
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

func TestErrors_CreateUser(t *testing.T) {
	mockDB := &MockDB{
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "", fmt.Errorf("db error")
		},
	}
	server := NewServer(mockDB, nil)
	token, _ := GenerateToken("admin", true)

	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(`{"username":"u", "password":"p", "display_name": "d"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

func TestErrors_ChangePassword(t *testing.T) {
	mockDB := &MockDB{
        VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
            return nil
        },
		UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
			return fmt.Errorf("db error")
		},
	}
	server := NewServer(mockDB, nil)
	token, _ := GenerateToken("user", false)

	req := httptest.NewRequest("POST", "/api/change-password", strings.NewReader(`{"old_password":"o", "new_password":"n"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

func TestValidation_Sync_Null_Content(t *testing.T) {
     mockDB := &MockDB{
        SyncFn: func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
            return []*RoomResult{}, []*MsgResult{}, []*UserResult{}, nil
        },
    }
    server := NewServer(mockDB, nil)
    userToken, _ := GenerateToken("user", false)

    // Explicit null content
    body := `{"messages": [{"room_id": "r", "message_id": 1, "content": null}]}`
    req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+userToken)
    w := httptest.NewRecorder()
    server.router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("Expected 200 for null content, got %d", w.Code)
    }
}
