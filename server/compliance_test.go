package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateUser_MissingDisplayName(t *testing.T) {
	mockDB := new(MockDB)
	mockDB.CreateUserFn = func(ctx context.Context, user CreateUserReq) (string, error) {
		return "new-user-id", nil
	}
	server := NewServer(mockDB, nil)

	// Admin Token
	token, _ := GenerateToken("admin", true)

	reqBody := CreateUserReq{
		Username: "newuser",
		Password: "password",
		// DisplayName is missing
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	// Expect 400 Bad Request because DisplayName is required
	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 Bad Request for missing display_name")
}

func TestLogin_MissingFields(t *testing.T) {
	mockDB := new(MockDB)
	server := NewServer(mockDB, nil)

	tests := []struct {
		name     string
		body     LoginRequest
		wantCode int
	}{
		{
			name:     "Missing Username",
			body:     LoginRequest{Password: "pass"},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Missing Password",
			body:     LoginRequest{Username: "user"},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestRoomOperations_ResponseBody(t *testing.T) {
	mockDB := new(MockDB)
	mockDB.DeleteRoomFn = func(ctx context.Context, id string) error { return nil }
	mockDB.RenameRoomFn = func(ctx context.Context, id, name string) error { return nil }
	server := NewServer(mockDB, nil)
	token, _ := GenerateToken("admin", true)

	t.Run("Delete Room Response", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/rooms/r1", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"status": "ok"}`, w.Body.String())
	})

	t.Run("Rename Room Response", func(t *testing.T) {
		body := `{"name": "New Name"}`
		req := httptest.NewRequest("PUT", "/api/rooms/r1", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"status": "ok"}`, w.Body.String())
	})
}

func TestSync_NullLastSyncedAt(t *testing.T) {
	mockDB := new(MockDB)
	mockDB.SyncFn = func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
		// Verify that lastSync is zero time (beginning of time)
		if !lastSync.IsZero() {
			t.Errorf("Expected zero time for lastSync, got %v", lastSync)
		}
		return []*RoomResult{}, []*MsgResult{}, []*UserResult{}, nil
	}
	server := NewServer(mockDB, nil)
	token, _ := GenerateToken("user", false)

	// JSON with null last_synced_at
	body := `{"last_synced_at": null}`
	req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
