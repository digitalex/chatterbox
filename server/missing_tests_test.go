package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateProfileMissingFields(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)

	// Missing public_key
	reqBody := `{"display_name": "New Name"}`
	req, _ := http.NewRequest("POST", "/api/me", strings.NewReader(reqBody))
	token, _ := GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("missing public_key: handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Missing display_name
	reqBody = `{"public_key": "key123"}`
	req, _ = http.NewRequest("POST", "/api/me", strings.NewReader(reqBody))
	token, _ = GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("missing display_name: handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestSyncInvalidNested(t *testing.T) {
	srv := NewServer(&MockDB{}, nil)

	// Invalid room (missing name)
	reqBody := `{
		"rooms": [{"room_id": "r1"}],
		"messages": []
	}`
	req, _ := http.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	token, _ := GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("invalid room: handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Invalid message (missing room_id)
	reqBody = `{
		"rooms": [],
		"messages": [{"message_id": 1}]
	}`
	req, _ = http.NewRequest("POST", "/api/sync", strings.NewReader(reqBody))
	token, _ = GenerateToken("test-user", false)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("invalid message: handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}
