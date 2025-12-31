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

// Helper to create a request with a token
func newAuthRequest(method, url, token string, body interface{}) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestValidation_Login(t *testing.T) {
	// Mock that fails authentication
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			return "", false, fmt.Errorf("invalid credentials")
		},
	}
	srv := NewServer(mockDB, nil)

	tests := []struct {
		name     string
		body     LoginRequest
		wantCode int
	}{
		{"Missing Username", LoginRequest{Password: "pass"}, http.StatusBadRequest},
		{"Missing Password", LoginRequest{Username: "user"}, http.StatusBadRequest},
		{"Empty Body", LoginRequest{}, http.StatusBadRequest},
		{"Valid", LoginRequest{Username: "user", Password: "pass"}, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/login", bytes.NewReader(body))
			w := httptest.NewRecorder()
			srv.router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("got code %d, want %d", w.Code, tt.wantCode)
			}
		})
	}
}

func TestValidation_CreateUser(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)
	token, _ := GenerateToken("admin", true)

	tests := []struct {
		name     string
		body     CreateUserReq
		wantCode int
	}{
		{"Missing Username", CreateUserReq{Password: "p", DisplayName: "d"}, http.StatusBadRequest},
		{"Missing Password", CreateUserReq{Username: "u", DisplayName: "d"}, http.StatusBadRequest},
		{"Missing DisplayName", CreateUserReq{Username: "u", Password: "p"}, http.StatusBadRequest}, // This is expected to fail currently
		{"Valid", CreateUserReq{Username: "u", Password: "p", DisplayName: "d"}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAuthRequest("POST", "/api/users", token, tt.body)
			w := httptest.NewRecorder()
			srv.router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("got code %d, want %d", w.Code, tt.wantCode)
			}
		})
	}
}

func TestValidation_Sync(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)
	token, _ := GenerateToken("user", false)

	// Custom struct to allow sending invalid data (missing required fields)
	type InvalidRoomReq struct {
		Name string `json:"name"` // Missing room_id
	}
	type SyncReqInvalidRoom struct {
		Rooms []InvalidRoomReq `json:"rooms"`
	}

	type InvalidMsgReq struct {
		RoomID string `json:"room_id"` // Missing message_id
	}
	type SyncReqInvalidMsg struct {
		Messages []InvalidMsgReq `json:"messages"`
	}

	t.Run("Invalid Room", func(t *testing.T) {
		body := SyncReqInvalidRoom{Rooms: []InvalidRoomReq{{Name: "test"}}}
		req := newAuthRequest("POST", "/api/sync", token, body)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got code %d, want %d for invalid room", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("Invalid Message", func(t *testing.T) {
		body := SyncReqInvalidMsg{Messages: []InvalidMsgReq{{RoomID: "r1"}}}
		req := newAuthRequest("POST", "/api/sync", token, body)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got code %d, want %d for invalid message", w.Code, http.StatusBadRequest)
		}
	})

    t.Run("Malformed JSON", func(t *testing.T) {
        req := httptest.NewRequest("POST", "/api/sync", bytes.NewReader([]byte(`{"rooms": [{"nam"`))) // Incomplete JSON
        req.Header.Set("Authorization", "Bearer "+token)
        w := httptest.NewRecorder()
        srv.router.ServeHTTP(w, req)

        if w.Code != http.StatusBadRequest {
            t.Errorf("got code %d, want %d for malformed JSON", w.Code, http.StatusBadRequest)
        }
    })
}

func TestValidation_ChangePassword(t *testing.T) {
	// Mock that fails password verification
	mockDB := &MockDB{
		VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
			return fmt.Errorf("invalid password")
		},
	}
	srv := NewServer(mockDB, nil)
	token, _ := GenerateToken("user", false)

	tests := []struct {
		name     string
		body     ChangePasswordReq
		wantCode int
	}{
		{"Missing OldPassword", ChangePasswordReq{NewPassword: "new"}, http.StatusBadRequest},
		{"Missing NewPassword", ChangePasswordReq{OldPassword: "old"}, http.StatusBadRequest},
		{"Valid", ChangePasswordReq{OldPassword: "old", NewPassword: "new"}, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAuthRequest("POST", "/api/change-password", token, tt.body)
			w := httptest.NewRecorder()
			srv.router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("got code %d, want %d", w.Code, tt.wantCode)
			}
		})
	}
}
