package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAPICompliance(t *testing.T) {
	// GET /
	t.Run("GET / Root Check", func(t *testing.T) {
		mockDB := &MockDB{}
		srv := NewServer(mockDB, nil)

		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "Chatterbox API is running 🚀", rr.Body.String())
	})

	// GET /health
	t.Run("GET /health", func(t *testing.T) {
		mockDB := &MockDB{
			HealthCheckFn: func(ctx context.Context) (int64, error) {
				return 1, nil
			},
		}
		srv := NewServer(mockDB, nil)

		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "ok", resp["status"])
		assert.EqualValues(t, 1, resp["db_check"])
	})

	// POST /api/sync
	t.Run("POST /api/sync", func(t *testing.T) {
		mockDB := &MockDB{
			SyncFn: func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
				return []*RoomResult{{RoomID: "r1", Name: "Room 1"}},
				       []*MsgResult{},
					   []*UserResult{}, nil
			},
		}
		srv := NewServer(mockDB, nil)
		token, _ := GenerateToken("u1", false)

		reqBody := SyncRequest{
			Rooms: []RoomReq{{RoomID: "r1", Name: "Room 1"}},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/sync", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp SyncResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotZero(t, resp.SyncTimestamp)
		assert.Len(t, resp.Rooms, 1)
		assert.Equal(t, "r1", resp.Rooms[0].RoomID)
	})

	// POST /api/sync 401
	t.Run("POST /api/sync Unauthorized", func(t *testing.T) {
		srv := NewServer(&MockDB{}, nil)
		req := httptest.NewRequest("POST", "/api/sync", nil) // No header
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	// POST /api/me
	t.Run("POST /api/me", func(t *testing.T) {
		mockDB := &MockDB{
			UpdateProfileFn: func(ctx context.Context, userID string, displayName *string, publicKey *string) error {
				assert.Equal(t, "u1", userID)
				assert.Equal(t, "New Name", *displayName)
				return nil
			},
		}
		srv := NewServer(mockDB, nil)
		token, _ := GenerateToken("u1", false)

		name := "New Name"
		reqBody := ProfileReq{DisplayName: &name}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/me", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	// POST /api/me 400
	t.Run("POST /api/me BadRequest", func(t *testing.T) {
		srv := NewServer(&MockDB{}, nil)
		token, _ := GenerateToken("u1", false)
		req := httptest.NewRequest("POST", "/api/me", bytes.NewBuffer([]byte(`{invalid-json}`)))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	// DELETE /api/rooms/{roomID}
	t.Run("DELETE /api/rooms/{roomID}", func(t *testing.T) {
		mockDB := &MockDB{
			DeleteRoomFn: func(ctx context.Context, roomID string) error {
				assert.Equal(t, "r1", roomID)
				return nil
			},
		}
		srv := NewServer(mockDB, nil)
		// Admin token
		token, _ := GenerateToken("admin", true)

		req := httptest.NewRequest("DELETE", "/api/rooms/r1", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		// Need to route through chi to get URL params working in test without explicit context hacking if possible,
		// but since we test handlers usually directly or via srv.router.ServeHTTP, ServeHTTP is better.
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp map[string]string
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Equal(t, "ok", resp["status"])
	})

	// DELETE /api/rooms/{roomID} 403
	t.Run("DELETE /api/rooms/{roomID} Forbidden", func(t *testing.T) {
		srv := NewServer(&MockDB{}, nil)
		token, _ := GenerateToken("user", false) // Not admin

		req := httptest.NewRequest("DELETE", "/api/rooms/r1", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	// PUT /api/rooms/{roomID}
	t.Run("PUT /api/rooms/{roomID}", func(t *testing.T) {
		mockDB := &MockDB{
			RenameRoomFn: func(ctx context.Context, roomID, name string) error {
				assert.Equal(t, "r1", roomID)
				assert.Equal(t, "New Room Name", name)
				return nil
			},
		}
		srv := NewServer(mockDB, nil)
		token, _ := GenerateToken("admin", true)

		body, _ := json.Marshal(map[string]string{"name": "New Room Name"})
		req := httptest.NewRequest("PUT", "/api/rooms/r1", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp map[string]string
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Equal(t, "ok", resp["status"])
	})

	// PUT /api/rooms/{roomID} 400
	t.Run("PUT /api/rooms/{roomID} BadRequest", func(t *testing.T) {
		srv := NewServer(&MockDB{}, nil)
		token, _ := GenerateToken("admin", true)

		// Missing name
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest("PUT", "/api/rooms/r1", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	// GET /api/rooms/{roomID}/members
	t.Run("GET /api/rooms/{roomID}/members", func(t *testing.T) {
		mockDB := &MockDB{
			GetRoomMembersFn: func(ctx context.Context, roomID string) ([]*RoomMember, error) {
				return []*RoomMember{{UserID: "u1", PublicKey: "k1"}}, nil
			},
		}
		srv := NewServer(mockDB, nil)

		req := httptest.NewRequest("GET", "/api/rooms/r1/members", nil)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp []RoomMember
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp, 1)
		assert.Equal(t, "u1", resp[0].UserID)
	})

	// POST /api/login
	t.Run("POST /api/login", func(t *testing.T) {
		mockDB := &MockDB{
			AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
				if username == "user" && password == "pass" {
					return "u1", false, nil
				}
				return "", false, status.Error(codes.Unauthenticated, "invalid credentials")
			},
		}
		srv := NewServer(mockDB, nil)

		reqBody := LoginRequest{Username: "user", Password: "pass"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp LoginResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
	})

	// POST /api/login 401
	t.Run("POST /api/login Unauthorized", func(t *testing.T) {
		mockDB := &MockDB{
			AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
				return "", false, status.Error(codes.Unauthenticated, "invalid credentials")
			},
		}
		srv := NewServer(mockDB, nil)

		reqBody := LoginRequest{Username: "user", Password: "wrong"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	// POST /api/users
	t.Run("POST /api/users", func(t *testing.T) {
		mockDB := &MockDB{
			CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
				return "new-id", nil
			},
		}
		srv := NewServer(mockDB, nil)
		token, _ := GenerateToken("admin", true)

		reqBody := CreateUserReq{Username: "u", Password: "p", DisplayName: "d"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp map[string]string
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Equal(t, "new-id", resp["user_id"])
	})

	// POST /api/users 403
	t.Run("POST /api/users Forbidden", func(t *testing.T) {
		srv := NewServer(&MockDB{}, nil)
		token, _ := GenerateToken("user", false) // Not admin

		reqBody := CreateUserReq{Username: "u", Password: "p", DisplayName: "d"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	// POST /api/change-password
	t.Run("POST /api/change-password", func(t *testing.T) {
		mockDB := &MockDB{
			VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
				return nil
			},
			UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
				return nil
			},
		}
		srv := NewServer(mockDB, nil)
		token, _ := GenerateToken("u1", false)

		reqBody := ChangePasswordReq{OldPassword: "old", NewPassword: "new"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp map[string]string
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Equal(t, "ok", resp["status"])
	})

	// POST /api/change-password 400 (missing new password)
	t.Run("POST /api/change-password BadRequest", func(t *testing.T) {
		srv := NewServer(&MockDB{}, nil)
		token, _ := GenerateToken("u1", false)

		reqBody := ChangePasswordReq{OldPassword: "old", NewPassword: ""}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
