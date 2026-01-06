package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSyncHandlersEdgeCases(t *testing.T) {
	mockDB := &MockDB{}
	server := NewServer(mockDB, nil)

	t.Run("Sync Bad Request - Invalid JSON", func(t *testing.T) {
		token, _ := GenerateToken("user1", false)
		req := httptest.NewRequest("POST", "/api/sync", bytes.NewBufferString(`{"rooms": `)) // Invalid JSON
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Sync Bad Request - Invalid Room", func(t *testing.T) {
		token, _ := GenerateToken("user1", false)
		req := httptest.NewRequest("POST", "/api/sync", bytes.NewBufferString(`{"rooms": [{"room_id": "r1"}]}`)) // Missing Name
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Sync Bad Request - Invalid Message", func(t *testing.T) {
		token, _ := GenerateToken("user1", false)
		req := httptest.NewRequest("POST", "/api/sync", bytes.NewBufferString(`{"messages": [{"room_id": "r1"}]}`)) // Missing MessageID (default 0)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})
}
