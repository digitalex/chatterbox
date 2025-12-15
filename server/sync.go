package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Server) createRoomHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	type RoomReq struct {
		Name string `json:"name"`
	}
	var req RoomReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Room name is required", http.StatusBadRequest)
		return
	}

	// Generate a UUID for the room
	roomID := uuid.New().String()

	err := s.db.CreateRoom(ctx, roomID, req.Name, userID)
	if err != nil {
		http.Error(w, "DB Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Return Success
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"room_id":    roomID,
		"name":       req.Name,
		"created_at": time.Now(), // Approximate, actual time is commit timestamp
	})
}

func (s *Server) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID, ok := UserIDFromContext(ctx)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req ProfileReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    err := s.db.UpdateProfile(ctx, userID, req.DisplayName, req.PublicKey)
    if err != nil {
        http.Error(w, "DB Error: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func (s *Server) getRoomMembersHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    roomID := chi.URLParam(r, "roomID")

    members, err := s.db.GetRoomMembers(ctx, roomID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(members)
}

func (s *Server) sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	roomID := chi.URLParam(r, "roomID")
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 1. Parse Request
	type MsgReq struct {
		Content interface{} `json:"content"` // Any JSON (text or encrypted blob)
	}
	var req MsgReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 2. Generate ID (Simple approach: Microseconds)
	msgID := time.Now().UnixMicro()

	err := s.db.SendMessage(ctx, roomID, userID, msgID, req.Content)
	if err != nil {
		http.Error(w, "DB Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Return Success
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message_id": msgID,
		"status":     "sent",
	})
}

func (s *Server) syncHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Parse Request Body
	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If body is empty, assume clean sync (req is zero-valued)
	}

	// Default to "beginning of time" if no timestamp provided
	lastSync := time.Time{}
	if req.LastSyncedAt != nil {
		lastSync = *req.LastSyncedAt
	}

	// Capture "Now" to return as the next cursor
	now := time.Now().UTC()

	rooms, messages, users, err := s.db.Sync(ctx, userID, lastSync)
    if err != nil {
        http.Error(w, "DB Error: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // 4. Return Response
    resp := SyncResponse{
        SyncTimestamp: now,
        Rooms:         rooms,
        Messages:      messages,
        Users:         users,
    }
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
