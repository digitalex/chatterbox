package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

func TestRootEndpoint(t *testing.T) {
	srv := NewServer(&MockDB{})

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
	srv := NewServer(mockDB)

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
		SyncFn: func(ctx context.Context, userID string, lastSync time.Time) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
			return []*RoomResult{{RoomID: "1", Name: "Test Room"}},
				[]*MsgResult{{MessageID: 1, Content: spanner.NullJSON{Value: "hello", Valid: true}}},
				[]*UserResult{{UserID: "u1", DisplayName: "User 1"}},
				nil
		},
	}
	srv := NewServer(mockDB)

	reqBody := `{"last_synced_at": "2023-01-01T00:00:00Z"}`
	req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	req.Header.Set("X-User-ID", "test-user")
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
}

func TestCreateRoom(t *testing.T) {
	mockDB := &MockDB{
		CreateRoomFn: func(ctx context.Context, roomID string, name string, userID string) error {
			if name != "New Room" {
				t.Errorf("Expected name 'New Room', got %s", name)
			}
			return nil
		},
	}
	srv := NewServer(mockDB)

	reqBody := `{"name": "New Room"}`
	req, _ := http.NewRequest("POST", "/api/rooms", strings.NewReader(reqBody))
	req.Header.Set("X-User-ID", "test-user")
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusCreated)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp["name"] != "New Room" {
		t.Errorf("Expected name 'New Room', got %v", resp["name"])
	}
	if _, ok := resp["room_id"]; !ok {
		t.Error("Expected room_id in response")
	}
}

func TestSendMessage(t *testing.T) {
	mockDB := &MockDB{
		SendMessageFn: func(ctx context.Context, roomID string, userID string, msgID int64, content interface{}) error {
			if roomID != "room-1" {
				t.Errorf("Expected roomID 'room-1', got %s", roomID)
			}
			return nil
		},
	}
	srv := NewServer(mockDB)

	reqBody := `{"content": "hello"}`
	req, _ := http.NewRequest("POST", "/api/rooms/room-1/messages", strings.NewReader(reqBody))
	req.Header.Set("X-User-ID", "test-user")

	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusCreated)
	}
}

func TestUpdateProfile(t *testing.T) {
	mockDB := &MockDB{
		UpdateProfileFn: func(ctx context.Context, userID string, displayName string, publicKey string) error {
			if displayName != "New Name" {
				t.Errorf("Expected DisplayName 'New Name', got %s", displayName)
			}
			return nil
		},
	}
	srv := NewServer(mockDB)

	reqBody := `{"display_name": "New Name", "public_key": "key123"}`
	req, _ := http.NewRequest("POST", "/api/me", strings.NewReader(reqBody))
	req.Header.Set("X-User-ID", "test-user")
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestUpdateProfile_MissingDisplayName(t *testing.T) {
	srv := NewServer(&MockDB{})

	reqBody := `{"public_key": "key123"}` // Missing display_name
	req, _ := http.NewRequest("POST", "/api/me", strings.NewReader(reqBody))
	req.Header.Set("X-User-ID", "test-user")
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for missing display_name: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestUpdateProfile_MissingPublicKey(t *testing.T) {
	srv := NewServer(&MockDB{})

	reqBody := `{"display_name": "test-user"}` // Missing public_key
	req, _ := http.NewRequest("POST", "/api/me", strings.NewReader(reqBody))
	req.Header.Set("X-User-ID", "test-user")
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for missing public_key: got %v want %v",
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
	srv := NewServer(mockDB)

	req, _ := http.NewRequest("GET", "/api/rooms/room-1/members", nil)
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
