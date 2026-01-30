package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompliance_CreateUser_MissingDisplayName(t *testing.T) {
	mockDB := &MockDB{
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "new-user-id", nil
		},
	}
	server := NewServer(mockDB, nil)

	adminToken, _ := GenerateToken("admin-id", true)
	// Missing display_name
	reqBody := `{"username": "newuser", "password": "password"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing display_name, got %d", w.Code)
	}
}

func TestCompliance_Sync_MalformedJSON(t *testing.T) {
	server := NewServer(&MockDB{}, nil)
	token, _ := GenerateToken("user-id", false)

	reqBody := `{"last_synced_at": "invalid-json`
	req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for malformed JSON, got %d", w.Code)
	}
}

func TestCompliance_Sync_InvalidRoom(t *testing.T) {
	server := NewServer(&MockDB{}, nil)
	token, _ := GenerateToken("user-id", false)

	// Missing room name
	reqBody := `{
		"rooms": [{"room_id": "r1"}]
	}`
	req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing room name, got %d", w.Code)
	}
}

func TestCompliance_Sync_InvalidMessage(t *testing.T) {
	server := NewServer(&MockDB{}, nil)
	token, _ := GenerateToken("user-id", false)

	// Missing message_id (defaults to 0, if we treat 0 as invalid it should fail)
	// But let's say missing room_id
	reqBody := `{
		"messages": [{"message_id": 123}]
	}`
	req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing message room_id, got %d", w.Code)
	}
}
