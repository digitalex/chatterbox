package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestComplianceCreateUser(t *testing.T) {
	mockDB := &MockDB{
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "new-user-id", nil
		},
	}
	srv := NewServer(mockDB, nil)

	// Missing DisplayName
	reqBody := `{"username": "testuser", "password": "password"}`
	req, _ := http.NewRequest("POST", "/api/users", strings.NewReader(reqBody))

	// Generate Admin Token (required for this endpoint)
	token, _ := GenerateToken("admin-id", true)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing display_name, got %v", status)
	}
}

func TestComplianceSyncMalformed(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)

	// Malformed JSON
	reqBody := `{"last_synced_at": "invalid-json`
	req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))

	token, _ := GenerateToken("user-id", false)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for malformed JSON, got %v", status)
	}
}

func TestComplianceSyncMissingFields(t *testing.T) {
	mockDB := &MockDB{
		SyncFn: func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
			return []*RoomResult{}, []*MsgResult{}, []*UserResult{}, nil
		},
	}
	srv := NewServer(mockDB, nil)

	// Valid JSON but missing required fields in nested objects
	reqBody := `{
		"rooms": [{"room_id": "r1"}],
		"messages": [{"room_id": "r1", "content": "hi"}]
	}`
	req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))

	token, _ := GenerateToken("user-id", false)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing nested fields, got %v", status)
	}
}
