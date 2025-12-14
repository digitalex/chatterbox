package main

import (
	"encoding/json"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

func (s *Server) createRoomHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.Header.Get("X-User-ID")

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

	// 1. Create the Room Mutation
	roomMutation := spanner.Insert("Rooms",
		[]string{"RoomId", "Name", "CreatedAt"},
		[]interface{}{roomID, req.Name, spanner.CommitTimestamp},
	)

	// 2. Create the Room Membership Mutation
	memberMutation := spanner.Insert("RoomMembers",
		[]string{"RoomId", "UserId", "JoinedAt", "LastReadMessageId"},
		[]interface{}{roomID, userID, spanner.CommitTimestamp, 0},
	)

	// 3. Apply BOTH mutations in one transaction
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{roomMutation, memberMutation})
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
    userID := r.Header.Get("X-User-ID")

    var req ProfileReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // 1. Create the User Mutation
    userMutation := spanner.InsertOrUpdate("Users",
        []string{"UserId", "DisplayName", "Email", "PublicKey", "CreatedAt"},
        []interface{}{userID, req.DisplayName, "anon", req.PublicKey, spanner.CommitTimestamp},
    )

    // 2. Create the Room Membership Mutation (Auto-Join General)
    // We use InsertOrUpdate so it doesn't crash if they are already a member.
    roomMutation := spanner.InsertOrUpdate("RoomMembers",
        []string{"RoomId", "UserId", "JoinedAt", "LastReadMessageId"},
        []interface{}{"room-general-001", userID, spanner.CommitTimestamp, 0},
    )

    // 3. Apply BOTH mutations in one transaction
    _, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{userMutation, roomMutation})
    if err != nil {
        http.Error(w, "DB Error: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func (s *Server) getRoomMembersHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    roomID := chi.URLParam(r, "roomID")

    stmt := spanner.Statement{
        SQL: `SELECT u.UserId, u.PublicKey 
              FROM RoomMembers rm
              JOIN Users u ON rm.UserId = u.UserId
              WHERE rm.RoomId = @rid`,
        Params: map[string]interface{}{"rid": roomID},
    }
    iter := s.spannerClient.Single().Query(ctx, stmt)
    defer iter.Stop()

    var members []*RoomMember
    for {
        row, err := iter.Next()
        if err == iterator.Done { break }
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        var m RoomMember
        // Handle Nullable PublicKey (older users might not have one)
        var pk spanner.NullString
        if err := row.Columns(&m.UserID, &pk); err != nil {
            http.Error(w, "Parse Error", http.StatusInternalServerError)
            return
        }
        m.PublicKey = pk.StringVal // Empty string if null
        members = append(members, &m)
    }

    json.NewEncoder(w).Encode(members)
}

func (s *Server) sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	roomID := chi.URLParam(r, "roomID")
	userID := r.Header.Get("X-User-ID")

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
	// In production, you might want a robust Snowflake ID generator.
	msgID := time.Now().UnixMicro()

	// 3. Spanner Mutation
	m := spanner.Insert("Messages",
		[]string{"RoomId", "MessageId", "SenderId", "Content", "CreatedAt"},
		[]interface{}{roomID, msgID, userID, spanner.NullJSON{Value: req.Content, Valid: true}, spanner.CommitTimestamp},
	)

	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{m})
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
	
	// 1. Auth Stub: In real life, get this drom a JWT middleware.
	// For now, we trust the header for testing.
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing X-User-ID header", http.StatusUnauthorized)
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

	// 3. Query Spanner
	// We need two things: The User's Room states, and the New Messages.
	// We can do this in parallel, but sequential is easier to read.

	// A. Get Rooms & Read State
	// "Get all rooms I am in, and where I left off."
	roomIter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT r.RoomId, r.Name, rm.LastReadMessageId 
              FROM RoomMembers rm
              JOIN Rooms r ON rm.RoomId = r.RoomId
              WHERE rm.UserId = @uid`,
		Params: map[string]interface{}{"uid": userID},
	})
	defer roomIter.Stop()

	var rooms []*RoomResult
	// We'll also build a list of RoomIDs to filter messages efficiently
	var myRoomIDs []string

	for {
		row, err := roomIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, "DB Error Rooms: "+err.Error(), http.StatusInternalServerError)
			return
		}
		var r RoomResult
		// Handle nullable LastReadMessageId (it might be null if they never read anything)
		var lastRead spanner.NullInt64
		if err := row.Columns(&r.RoomID, &r.Name, &lastRead); err != nil {
			http.Error(w, "Parse Error Rooms", http.StatusInternalServerError)
			return
		}
		r.LastReadMessageID = lastRead.Int64 // Defaults to 0 if null
		rooms = append(rooms, &r)
		myRoomIDs = append(myRoomIDs, r.RoomID)
	}

	// B. Get New Messages
	// "Get messages created after LastSync, belonging to my rooms."
	// Note: If myRoomIDs is empty, we skip this to avoid SQL error.
	var messages []*MsgResult
	if len(myRoomIDs) > 0 {
		stmt := spanner.Statement{
			SQL: `SELECT RoomId, MessageId, SenderId, Content, CreatedAt
			      FROM Messages@{FORCE_INDEX=MessagesByTime}
			      WHERE CreatedAt > @since 
			      AND RoomId IN UNNEST(@rooms)
			      ORDER BY CreatedAt ASC`,
			Params: map[string]interface{}{
				"since": lastSync,
				"rooms": myRoomIDs,
			},
		}
		msgIter := s.spannerClient.Single().Query(ctx, stmt)
		defer msgIter.Stop()

		for {
			row, err := msgIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "DB Error Msgs: "+err.Error(), http.StatusInternalServerError)
				return
			}
			var m MsgResult
			if err := row.Columns(&m.RoomID, &m.MessageID, &m.SenderID, &m.Content, &m.CreatedAt); err != nil {
				http.Error(w, "Parse Error Msgs", http.StatusInternalServerError)
				return
			}
			messages = append(messages, &m)
		}
	}

	var users []*UserResult
    if len(messages) > 0 {
        // 1. Collect unique sender IDs
        uniqueUsers := make(map[string]bool)
        var userIDs []string
        for _, m := range messages {
            if !uniqueUsers[m.SenderID] {
                uniqueUsers[m.SenderID] = true
                userIDs = append(userIDs, m.SenderID)
            }
        }

        // 2. Query Users table
        stmt := spanner.Statement{
            SQL: `SELECT UserId, DisplayName 
                  FROM Users 
                  WHERE UserId IN UNNEST(@uids)`,
            Params: map[string]interface{}{"uids": userIDs},
        }
        userIter := s.spannerClient.Single().Query(ctx, stmt)
        defer userIter.Stop()

        for {
            row, err := userIter.Next()
            if err == iterator.Done { break }
            if err != nil {
                http.Error(w, "DB Error Users: "+err.Error(), http.StatusInternalServerError)
                return
            }
            var u UserResult
            if err := row.Columns(&u.UserID, &u.DisplayName); err != nil {
                // If DisplayName is missing/null, handle gracefully or ignore
                continue 
            }
            users = append(users, &u)
        }
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

