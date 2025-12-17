package main

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
)

type UserResult struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
}

// API Request/Response shapes
type SyncRequest struct {
	LastSyncedAt *time.Time `json:"last_synced_at"` // Nullable for first load
	Rooms        []RoomReq  `json:"rooms"`
	Messages     []MsgReq   `json:"messages"`
}

type SyncResponse struct {
	SyncTimestamp time.Time     `json:"sync_timestamp"`
	Rooms         []*RoomResult `json:"rooms"`
	Messages      []*MsgResult  `json:"messages"`
	Users         []*UserResult `json:"users"`
}

// Input structures for SyncRequest
type RoomReq struct {
	RoomID string `json:"room_id"` // Client generated UUID
	Name   string `json:"name"`
}

type MsgReq struct {
	RoomID    string      `json:"room_id"`
	MessageID int64       `json:"message_id"`
	Content   interface{} `json:"content"` // Any JSON
}

// Data structures for JSON response
type RoomResult struct {
	RoomID            string `json:"room_id"`
	Name              string `json:"name"`
	LastReadMessageID int64  `json:"last_read_message_id"`
}

type MsgResult struct {
	RoomID    string           `json:"room_id"`
	MessageID int64            `json:"message_id"`
	SenderID  string           `json:"sender_id"`
	Content   spanner.NullJSON `json:"content"` // Handles the E2EE JSON blob
	CreatedAt time.Time        `json:"created_at"`
}

// Update ProfileReq
type ProfileReq struct {
	DisplayName string `json:"display_name"`
	PublicKey   string `json:"public_key"` // New field
}

// Add a Response struct for the new endpoint
type RoomMember struct {
	UserID    string `json:"user_id"`
	PublicKey string `json:"public_key"`
}

// Database interface abstracts the data store interactions
type Database interface {
	HealthCheck(ctx context.Context) (int64, error)
	UpdateProfile(ctx context.Context, userID string, displayName string, publicKey string) error
	GetRoomMembers(ctx context.Context, roomID string) ([]*RoomMember, error)
	Sync(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error)
}
