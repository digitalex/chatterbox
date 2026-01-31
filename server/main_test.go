package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

func TestRootEndpoint(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)

	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "Chatterbox API is running 🚀"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestHealthCheck(t *testing.T) {
	mockDB := &MockDB{
		HealthCheckFn: func(ctx context.Context) (int64, error) {
			return 1, nil
		},
	}
	srv := NewServer(mockDB, nil)

	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Expected status ok, got %v", resp["status"])
	}
	if val, ok := resp["db_check"].(float64); !ok || val != 1 {
		t.Errorf("Expected db_check 1, got %v", resp["db_check"])
	}
}

func TestSync(t *testing.T) {
	mockDB := &MockDB{
		SyncFn: func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
			return []*RoomResult{{RoomID: "1", Name: "Test Room"}},
				[]*MsgResult{{MessageID: 1, Content: spanner.NullJSON{Value: "hello", Valid: true}}},
				[]*UserResult{{UserID: "u1", DisplayName: "User 1"}},
				nil
		},
	}
	srv := NewServer(mockDB, nil)

	reqBody := `{"last_synced_at": "2023-01-01T00:00:00Z"}`
	req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	token, _ := GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var resp SyncResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Rooms) != 1 || resp.Rooms[0].RoomID != "1" {
		t.Errorf("Unexpected rooms response")
	}

	if len(resp.Messages) != 1 || resp.Messages[0].MessageID != 1 {
		t.Errorf("Unexpected messages response")
	}

	if len(resp.Users) != 1 || resp.Users[0].UserID != "u1" {
		t.Errorf("Unexpected users response")
	}
}

func TestSyncUpstream(t *testing.T) {
	mockDB := &MockDB{
		SyncFn: func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
			if len(rooms) != 1 || rooms[0].Name != "New Room" {
				t.Errorf("Expected 1 room named 'New Room'")
			}
			if len(messages) != 1 || messages[0].Content.(string) != "Hello" {
				t.Errorf("Expected 1 message with content 'Hello'")
			}
			return []*RoomResult{}, []*MsgResult{}, []*UserResult{}, nil
		},
	}
	srv := NewServer(mockDB, nil)

	reqBody := `{
		"rooms": [{"room_id": "r1", "name": "New Room"}],
		"messages": [{"room_id": "r1", "message_id": 1, "content": "Hello"}]
	}`
	req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	token, _ := GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestSyncUnauthorized(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)
	req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusUnauthorized)
	}
}

func TestSyncMalformedJSON(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)
	req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(`{"last_synced_at":`)) // Broken JSON
	token, _ := GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestUpdateProfile(t *testing.T) {
	mockDB := &MockDB{
		UpdateProfileFn: func(ctx context.Context, userID string, displayName *string, publicKey *string) error {
			if displayName == nil || *displayName != "New Name" {
				t.Errorf("Expected DisplayName 'New Name', got %v", displayName)
			}
			return nil
		},
	}
	srv := NewServer(mockDB, nil)

	reqBody := `{"display_name": "New Name", "public_key": "key123"}`
	req, _ := http.NewRequest("POST", "/api/me", strings.NewReader(reqBody))
	token, _ := GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if rr.Body.Len() > 0 {
		t.Errorf("Expected empty body, got %s", rr.Body.String())
	}
}

func TestUpdateProfileBadRequest(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)
	req, _ := http.NewRequest("POST", "/api/me", strings.NewReader(`{"display_name":`)) // Invalid JSON
	token, _ := GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestGetRoomMembers(t *testing.T) {
	mockDB := &MockDB{
		GetRoomMembersFn: func(ctx context.Context, roomID string) ([]*RoomMember, error) {
			if roomID != "room-1" {
				t.Errorf("Expected roomID 'room-1', got %s", roomID)
			}
			return []*RoomMember{{UserID: "u1", PublicKey: "key1"}}, nil
		},
	}
	srv := NewServer(mockDB, nil)

	req, _ := http.NewRequest("GET", "/api/rooms/room-1/members", nil)
	token, _ := GenerateToken("u1", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var members []*RoomMember
	if err := json.Unmarshal(rr.Body.Bytes(), &members); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(members) != 1 || members[0].UserID != "u1" {
		t.Errorf("Unexpected members response")
	}
	if members[0].PublicKey != "key1" {
		t.Errorf("Expected PublicKey 'key1', got %s", members[0].PublicKey)
	}
}

func TestJSONMarshaling(t *testing.T) {
	now := time.Date(2025, 10, 27, 10, 0, 0, 0, time.UTC)
	
	resp := SyncResponse{
		SyncTimestamp: now,
		Rooms: []*RoomResult{
			{RoomID: "1", Name: "Test Room", LastReadMessageID: 5},
		},
		Messages: []*MsgResult{},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshaling failed: %v", err)
	}

	jsonStr := string(data)
	
	// Check for snake_case keys
	if !strings.Contains(jsonStr, "sync_timestamp") {
		t.Error("JSON missing 'sync_timestamp' key")
	}
	if !strings.Contains(jsonStr, "last_read_message_id") {
		t.Error("JSON missing 'last_read_message_id' key")
	}
}

// Test 500 errors
func TestHealthCheckError(t *testing.T) {
	mockDB := &MockDB{
		HealthCheckFn: func(ctx context.Context) (int64, error) {
			return 0, fmt.Errorf("DB error")
		},
	}
	srv := NewServer(mockDB, nil)
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}
}

func TestSyncError(t *testing.T) {
	mockDB := &MockDB{
		SyncFn: func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
			return nil, nil, nil, fmt.Errorf("DB error")
		},
	}
	srv := NewServer(mockDB, nil)
	req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(`{}`))
	token, _ := GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}
}

func TestUpdateProfileError(t *testing.T) {
	mockDB := &MockDB{
		UpdateProfileFn: func(ctx context.Context, userID string, displayName *string, publicKey *string) error {
			return fmt.Errorf("DB error")
		},
	}
	srv := NewServer(mockDB, nil)
	req, _ := http.NewRequest("POST", "/api/me", strings.NewReader(`{"display_name": "error", "public_key": "error"}`))
	token, _ := GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}
}

func TestGetRoomMembersError(t *testing.T) {
	mockDB := &MockDB{
		GetRoomMembersFn: func(ctx context.Context, roomID string) ([]*RoomMember, error) {
			return nil, fmt.Errorf("DB error")
		},
	}
	srv := NewServer(mockDB, nil)
	req, _ := http.NewRequest("GET", "/api/rooms/r1/members", nil)
	token, _ := GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}
}
