package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Compliance tests to ensure implementation matches OpenAPI spec validation rules

func TestLogin_Compliance(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "Missing Username",
			body:     `{"password": "password"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Missing Password",
			body:     `{"username": "user"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Empty Fields",
			body:     `{"username": "", "password": ""}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/api/login", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()
			srv.router.ServeHTTP(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("loginHandler returned wrong status code: got %v want %v",
					rr.Code, tt.wantCode)
			}
		})
	}
}

func TestCreateUser_Compliance(t *testing.T) {
	mockDB := &MockDB{
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "new-id", nil
		},
	}
	srv := NewServer(mockDB, nil)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "Missing DisplayName",
			body:     `{"username": "user", "password": "pass"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Missing Username",
			body:     `{"display_name": "User", "password": "pass"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Missing Password",
			body:     `{"username": "user", "display_name": "User"}`,
			wantCode: http.StatusBadRequest,
		},
	}

	adminToken, _ := GenerateToken("admin", true)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/api/users", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+adminToken)
			rr := httptest.NewRecorder()
			srv.router.ServeHTTP(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("createUserHandler returned wrong status code: got %v want %v",
					rr.Code, tt.wantCode)
			}
		})
	}
}

func TestChangePassword_Compliance(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)
	userToken, _ := GenerateToken("user", false)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "Missing OldPassword",
			body:     `{"new_password": "new"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Missing NewPassword",
			body:     `{"old_password": "old"}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/api/change-password", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+userToken)
			rr := httptest.NewRecorder()
			srv.router.ServeHTTP(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("changePasswordHandler returned wrong status code: got %v want %v",
					rr.Code, tt.wantCode)
			}
		})
	}
}

func TestSync_Compliance(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)
	userToken, _ := GenerateToken("user", false)

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(`{invalid-json`))
		req.Header.Set("Authorization", "Bearer "+userToken)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %v", rr.Code)
		}
	})

	t.Run("Missing Room Required Fields", func(t *testing.T) {
		// Missing name
		body := `{
			"rooms": [{"room_id": "r1"}]
		}`
		req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing room name, got %v", rr.Code)
		}
	})

	t.Run("Missing Message Required Fields", func(t *testing.T) {
		// Missing message_id (zero value 0)
		body := `{
			"messages": [{"room_id": "r1", "content": "hi"}]
		}`
		req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing message_id, got %v", rr.Code)
		}

		// Missing room_id
		body2 := `{
			"messages": [{"message_id": 123, "content": "hi"}]
		}`
		req2, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(body2))
		req2.Header.Set("Authorization", "Bearer "+userToken)
		rr2 := httptest.NewRecorder()
		srv.router.ServeHTTP(rr2, req2)

		if rr2.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing message room_id, got %v", rr2.Code)
		}
	})
}
