package main

import (
	"time"

	"cloud.google.com/go/spanner"
)

// API Request/Response shapes
type SyncRequest struct {
	LastSyncedAt *time.Time `json:"last_synced_at"` // Nullable for first load
}

type SyncResponse struct {
	SyncTimestamp time.Time     `json:"sync_timestamp"`
	Rooms         []*RoomResult `json:"rooms"`
	Messages      []*MsgResult  `json:"messages"`
}

// Data structures for JSON response
type RoomResult struct {
	RoomID           string `json:"room_id"`
	Name             string `json:"name"`
	LastReadMessageID int64  `json:"last_read_message_id"`
}

type MsgResult struct {
	RoomID    string            `json:"room_id"`
	MessageID int64             `json:"message_id"`
	SenderID  string            `json:"sender_id"`
	Content   spanner.NullJSON  `json:"content"` // Handles the E2EE JSON blob
	CreatedAt time.Time         `json:"created_at"`
}

