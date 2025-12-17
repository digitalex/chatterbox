package main

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// Generate an invite code
func (s *Server) generateInviteHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID, ok := UserIDFromContext(ctx)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    roomID := chi.URLParam(r, "roomID")

    // Check if user is owner
    isOwner, err := s.db.IsRoomOwner(ctx, roomID, userID)
    if err != nil {
        http.Error(w, "DB Error: "+err.Error(), http.StatusInternalServerError)
        return
    }
    if !isOwner {
        http.Error(w, "Only room owner can generate invites", http.StatusForbidden)
        return
    }

    var req InviteRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        // Optional body
    }

    // Default expiration: 1 day
    duration := 24 * time.Hour
    if req.ExpiresInSeconds > 0 {
        duration = time.Duration(req.ExpiresInSeconds) * time.Second
    }
    expiresAt := time.Now().Add(duration)

    // Generate random code
    inviteCode := generateRandomCode(8)

    err = s.db.GenerateInvite(ctx, roomID, inviteCode, userID, expiresAt)
    if err != nil {
        http.Error(w, "DB Error: "+err.Error(), http.StatusInternalServerError)
        return
    }

    resp := InviteResponse{
        InviteCode: inviteCode,
        ExpiresAt:  expiresAt,
    }
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(resp)
}

// Accept an invite code
func (s *Server) acceptInviteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	inviteCode := chi.URLParam(r, "inviteCode")

	roomID, err := s.db.AcceptInvite(ctx, inviteCode, userID)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				http.Error(w, "Invite not found", http.StatusNotFound)
				return
			case codes.FailedPrecondition:
				http.Error(w, "Invite expired or already used", http.StatusPreconditionFailed)
				return
			}
		}
		http.Error(w, "Error accepting invite: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := AcceptInviteResponse{
		RoomID: roomID,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomCode(length int) string {
	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// Fallback or handle error? For this context, simple fallback or panic might be okay,
			// but better to just ignore as it is unlikely.
			// However, since we return string, we must return something.
			// Let's assume crypto/rand works.
			b[i] = charset[0]
		} else {
			b[i] = charset[num.Int64()]
		}
	}
	return string(b)
}
