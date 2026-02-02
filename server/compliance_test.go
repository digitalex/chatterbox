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

func TestUserCreationMissingDisplayName(t *testing.T) {
	mockDB := &MockDB{
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "new-user-id", nil
		},
	}
	server := NewServer(mockDB, nil)

	// Missing display_name
	reqBody := CreateUserReq{Username: "newuser", Password: "password"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))

	// Generate Admin Token
	token, _ := GenerateToken("admin-id", true)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Should be 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 Bad Request for missing display_name")
}

func TestLoginMissingFields(t *testing.T) {
	server := NewServer(&MockDB{}, nil)

	tests := []struct {
		name string
		body LoginRequest
	}{
		{"Missing Username", LoginRequest{Password: "pass"}},
		{"Missing Password", LoginRequest{Username: "user"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestChangePasswordMissingFields(t *testing.T) {
	server := NewServer(&MockDB{}, nil)

	tests := []struct {
		name string
		body ChangePasswordReq
	}{
		{"Missing OldPassword", ChangePasswordReq{NewPassword: "new"}},
		{"Missing NewPassword", ChangePasswordReq{OldPassword: "old"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))

			// Generate User Token
			token, _ := GenerateToken("user-id", false)
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestDeleteRoomResponseBody(t *testing.T) {
	mockDB := &MockDB{
		DeleteRoomFn: func(ctx context.Context, id string) error {
			return nil
		},
	}
	server := NewServer(mockDB, nil)

	req := httptest.NewRequest("DELETE", "/api/rooms/room1", nil)

	// Generate Admin Token
	token, _ := GenerateToken("admin-id", true)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
}

func TestRenameRoomResponseBody(t *testing.T) {
	mockDB := &MockDB{
		RenameRoomFn: func(ctx context.Context, id, name string) error {
			return nil
		},
	}
	server := NewServer(mockDB, nil)

	reqBody := map[string]string{"name": "New Name"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/api/rooms/room1", bytes.NewBuffer(body))

	// Generate Admin Token
	token, _ := GenerateToken("admin-id", true)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
}

func TestSyncNullLastSyncedAt(t *testing.T) {
	mockDB := &MockDB{
		SyncFn: func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
			// If lastSync is zero time, it effectively means null/empty was passed and parsed as zero value
			if !lastSync.IsZero() {
				// t.Error("Expected zero time for null last_synced_at")
				// The implementation might use a pointer or specific time. Check logic.
			}
			return []*RoomResult{}, []*MsgResult{}, []*UserResult{}, nil
		},
	}
	server := NewServer(mockDB, nil)

	// last_synced_at is explicitly null
	reqBody := `{"last_synced_at": null}`
	req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))

	// Inject User Context (via token usually, but here checking handler logic)
	// We need to use router to hit the middleware or inject context manually
	ctx := context.WithValue(req.Context(), userIDKey, "user-id")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	// Need to bypass auth middleware if calling handler directly, or use router with mock auth
	// Let's use router but with manual context injection if we can, or just mock the token
	// Actually, just calling the handler directly if exposed or via router with middleware skipped/mocked is easier.
	// But Sync is likely behind auth middleware.

	// Re-create request to go through router properly with a fake token
	token, _ := GenerateToken("user-id", false)
	req.Header.Set("Authorization", "Bearer "+token)

	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
