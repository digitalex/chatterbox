package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings" // <--- Added this import
	"testing"
	"time"
)

func TestRootEndpoint(t *testing.T) {
	// Initialize server with nil client (safe for route that doesn't use DB)
	srv := NewServer(nil)

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

