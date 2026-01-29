package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

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
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate nested objects
	for _, room := range req.Rooms {
		if room.RoomID == "" || room.Name == "" {
			http.Error(w, "Room ID and Name are required", http.StatusBadRequest)
			return
		}
	}
	for _, msg := range req.Messages {
		if msg.RoomID == "" || msg.MessageID == 0 {
			http.Error(w, "Room ID and Message ID are required", http.StatusBadRequest)
			return
		}
	}

	// Default to "beginning of time" if no timestamp provided
	lastSync := time.Time{}
	if req.LastSyncedAt != nil {
		lastSync = *req.LastSyncedAt
	}

	// Capture "Now" to return as the next cursor
	now := time.Now().UTC()

	rooms, messages, users, err := s.db.Sync(ctx, userID, lastSync, req.Rooms, req.Messages)
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
