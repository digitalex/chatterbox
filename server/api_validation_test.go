package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIValidation(t *testing.T) {
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			return "user-id", false, nil
		},
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "new-user-id", nil
		},
		VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
			return nil
		},
		UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
			return nil
		},
	}
	server := NewServer(mockDB, nil)

	// Helper to create admin token
	adminToken, _ := GenerateToken("admin-id", true)
	// Helper to create user token
	userToken, _ := GenerateToken("user-id", false)

	t.Run("POST /api/login - Missing Fields", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{"Missing Username", `{"password": "pass"}`},
			{"Missing Password", `{"username": "user"}`},
			{"Empty Username", `{"username": "", "password": "pass"}`},
			{"Empty Password", `{"username": "user", "password": ""}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "/api/login", strings.NewReader(tt.body))
				w := httptest.NewRecorder()
				server.router.ServeHTTP(w, req)

				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected 400 Bad Request, got %d", w.Code)
				}
			})
		}
	})

	t.Run("POST /api/users - Missing Fields", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{"Missing DisplayName", `{"username": "u", "password": "p"}`},
			{"Empty DisplayName", `{"username": "u", "password": "p", "display_name": ""}`},
			{"Missing Username", `{"password": "p", "display_name": "d"}`},
			{"Missing Password", `{"username": "u", "display_name": "d"}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "/api/users", strings.NewReader(tt.body))
				req.Header.Set("Authorization", "Bearer "+adminToken)
				w := httptest.NewRecorder()
				server.router.ServeHTTP(w, req)

				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected 400 Bad Request, got %d", w.Code)
				}
			})
		}
	})

	t.Run("POST /api/change-password - Missing Fields", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{"Missing OldPassword", `{"new_password": "new"}`},
			{"Empty OldPassword", `{"old_password": "", "new_password": "new"}`},
			{"Missing NewPassword", `{"old_password": "old"}`},
			{"Empty NewPassword", `{"old_password": "old", "new_password": ""}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "/api/change-password", strings.NewReader(tt.body))
				req.Header.Set("Authorization", "Bearer "+userToken)
				w := httptest.NewRecorder()
				server.router.ServeHTTP(w, req)

				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected 400 Bad Request, got %d", w.Code)
				}
			})
		}
	})

	t.Run("POST /api/sync - Validation", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{"Room Missing Name", `{"rooms": [{"room_id": "uuid"}]}`},
			{"Room Missing ID", `{"rooms": [{"name": "Room"}]}`},
			{"Msg Missing RoomID", `{"messages": [{"message_id": 1}]}`},
			{"Msg Missing MsgID", `{"messages": [{"room_id": "uuid"}]}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(tt.body))
				req.Header.Set("Authorization", "Bearer "+userToken)
				w := httptest.NewRecorder()
				server.router.ServeHTTP(w, req)

				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected 400 Bad Request, got %d", w.Code)
				}
			})
		}
	})

	t.Run("POST /api/sync - Empty Body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(`{invalid`))
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for invalid JSON, got %d", w.Code)
		}
	})
}
