package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

    "cloud.google.com/go/spanner"
)

func TestSyncHandlers(t *testing.T) {
	// Mock Data
	mockRoom := &RoomResult{RoomID: "room1", Name: "General", LastReadMessageID: 10}
	mockMsg := &MsgResult{
        RoomID: "room1",
        MessageID: 11,
        SenderID: "user2",
        Content: spanner.NullJSON{Value: "hello", Valid: true},
        CreatedAt: time.Now(),
    }
	mockUser := &UserResult{UserID: "user2", DisplayName: "Bob"}
    mockMember := &RoomMember{UserID: "user1", PublicKey: "key1"}

	mockDB := &MockDB{
		SyncFn: func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
			return []*RoomResult{mockRoom}, []*MsgResult{mockMsg}, []*UserResult{mockUser}, nil
		},
		UpdateProfileFn: func(ctx context.Context, userID string, displayName *string, publicKey *string) error {
			return nil
		},
        GetRoomMembersFn: func(ctx context.Context, roomID string) ([]*RoomMember, error) {
            if roomID == "room1" {
                return []*RoomMember{mockMember}, nil
            }
            return []*RoomMember{}, nil
        },
	}

	server := NewServer(mockDB, nil)
	token, _ := GenerateToken("user1", false)

	t.Run("Sync Success", func(t *testing.T) {
		reqBody := SyncRequest{
			Rooms:    []RoomReq{},
			Messages: []MsgReq{},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/sync", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp SyncResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(resp.Rooms) != 1 || resp.Rooms[0].RoomID != "room1" {
			t.Error("Expected room1 in response")
		}
	})

	t.Run("Update Profile Success", func(t *testing.T) {
        name := "Alice"
        key := "pubkey"
		reqBody := ProfileReq{DisplayName: &name, PublicKey: &key}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/me", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

    t.Run("Get Room Members Success", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/rooms/room1/members", nil)
		req.Header.Set("Authorization", "Bearer "+token)
        w := httptest.NewRecorder()
        server.router.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
            t.Errorf("Expected 200, got %d", w.Code)
        }

        var members []RoomMember
        if err := json.NewDecoder(w.Body).Decode(&members); err != nil {
             t.Fatalf("Failed to decode: %v", err)
        }
        if len(members) != 1 || members[0].UserID != "user1" {
            t.Error("Expected user1 member")
        }
    })
}
