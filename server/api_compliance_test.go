package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompliance_CreateUser_MissingDisplayName(t *testing.T) {
	// Setup
	mockDB := &MockDB{
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "new-id", nil
		},
	}
	server := NewServer(mockDB, nil)
	adminToken, _ := GenerateToken("admin", true)

	// Test Case: Missing display_name
	reqBody := `{"username": "testuser", "password": "password"}` // Missing display_name
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing display_name, got %d", w.Code)
	}
}

func TestCompliance_Login_MissingFields(t *testing.T) {
	server := NewServer(&MockDB{}, nil)

	// Test Case 1: Missing Username
	reqBody1 := `{"password": "password"}`
	req1 := httptest.NewRequest("POST", "/api/login", strings.NewReader(reqBody1))
	w1 := httptest.NewRecorder()
	server.router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing username, got %d", w1.Code)
	}

	// Test Case 2: Missing Password
	reqBody2 := `{"username": "user"}`
	req2 := httptest.NewRequest("POST", "/api/login", strings.NewReader(reqBody2))
	w2 := httptest.NewRecorder()
	server.router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing password, got %d", w2.Code)
	}
}

func TestCompliance_Sync_InvalidJSON(t *testing.T) {
	// Setup
	server := NewServer(&MockDB{}, nil)
	token, _ := GenerateToken("user", false)

	// Test Case: Invalid JSON
	reqBody := `{"rooms": [{"room_id": "1",` // Incomplete JSON
	req := httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for invalid JSON, got %d", w.Code)
	}
}

func TestCompliance_Sync_MissingRequiredFields(t *testing.T) {
	// Setup
	server := NewServer(&MockDB{
		SyncFn: func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
			return nil, nil, nil, nil
		},
	}, nil)
	token, _ := GenerateToken("user", false)

	// Test Case 1: Room missing name
	reqBody1 := `{"rooms": [{"room_id": "123"}]}`
	req1 := httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody1))
	req1.Header.Set("Authorization", "Bearer "+token)
	w1 := httptest.NewRecorder()

	server.router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for room missing name, got %d", w1.Code)
	}

	// Test Case 2: Room missing room_id
	reqBody2 := `{"rooms": [{"name": "Room Name"}]}`
	req2 := httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody2))
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()

	server.router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for room missing room_id, got %d", w2.Code)
	}

	// Test Case 3: Message missing room_id
	reqBody3 := `{"messages": [{"message_id": 1}]}`
	req3 := httptest.NewRequest("POST", "/api/sync", strings.NewReader(reqBody3))
	req3.Header.Set("Authorization", "Bearer "+token)
	w3 := httptest.NewRecorder()

	server.router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for message missing room_id, got %d", w3.Code)
	}

}
