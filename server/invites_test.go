package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateInvite(t *testing.T) {
	mockDB := &MockDB{
		IsRoomOwnerFn: func(ctx context.Context, roomID string, userID string) (bool, error) {
			if userID == "owner-user" {
				return true, nil
			}
			return false, nil
		},
		GenerateInviteFn: func(ctx context.Context, roomID string, inviteCode string, createdBy string, expiresAt time.Time) error {
			return nil
		},
	}

	server := NewServer(mockDB, nil)

	// Helper to generate a valid token for testing
	generateTestToken := func(userID string) string {
		token, _ := GenerateToken(userID)
		return token
	}

	// Test case: Success
	t.Run("Success", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"expires_in_seconds": 3600,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/rooms/room-1/invites", bytes.NewBuffer(body))

		// Authenticate
		token := generateTestToken("owner-user")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		server.router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}

		var resp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["invite_code"] == "" {
			t.Errorf("expected invite_code in response")
		}
	})

	// Test case: Forbidden
	t.Run("Forbidden", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/rooms/room-1/invites", nil)
		// Authenticate as other-user
		token := generateTestToken("other-user")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		server.router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusForbidden {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
		}
	})
}

func TestAcceptInvite(t *testing.T) {
	mockDB := &MockDB{
		AcceptInviteFn: func(ctx context.Context, inviteCode string, userID string) (string, error) {
			if inviteCode == "valid-code" {
				return "room-1", nil
			}
			return "", nil // In real DB this would error, but mock returns empty string for roomID if we don't return error
		},
	}

	server := NewServer(mockDB, nil)

	// Helper to generate a valid token for testing
	generateTestToken := func(userID string) string {
		token, _ := GenerateToken(userID)
		return token
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/invites/valid-code/accept", nil)
		// Authenticate
		token := generateTestToken("user-1")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		server.router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["room_id"] != "room-1" {
			t.Errorf("expected room_id to be room-1, got %v", resp["room_id"])
		}
	})
}
