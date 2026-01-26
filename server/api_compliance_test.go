package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAPICompliance(t *testing.T) {
	mockDB := &MockDB{
		AuthenticateUserFn: func(ctx context.Context, username, password string) (string, bool, error) {
			if username == "admin" && password == "password" {
				return "admin-id", true, nil
			}
			if username == "user" && password == "password" {
				return "user-id", false, nil
			}
			return "", false, status.Error(codes.Unauthenticated, "Invalid credentials")
		},
		CreateUserFn: func(ctx context.Context, user CreateUserReq) (string, error) {
			return "new-user-id", nil
		},
		DeleteRoomFn: func(ctx context.Context, roomID string) error {
			return nil
		},
		RenameRoomFn: func(ctx context.Context, roomID, newName string) error {
			return nil
		},
		VerifyPasswordFn: func(ctx context.Context, userID, password string) error {
			if password == "oldpass" {
				return nil
			}
			return status.Error(codes.Unauthenticated, "Invalid old password")
		},
		UpdatePasswordFn: func(ctx context.Context, userID, newPassword string) error {
			return nil
		},
	}

	server := NewServer(mockDB, nil)

	// Helper to create admin context
	adminCtx := func(req *http.Request) *http.Request {
		ctx := context.WithValue(req.Context(), userIDKey, "admin-id")
		ctx = context.WithValue(ctx, isAdminKey, true)
		return req.WithContext(ctx)
	}

	// Helper to create user context
	userCtx := func(req *http.Request) *http.Request {
		ctx := context.WithValue(req.Context(), userIDKey, "user-id")
		ctx = context.WithValue(ctx, isAdminKey, false)
		return req.WithContext(ctx)
	}

	t.Run("POST /api/users - Missing DisplayName", func(t *testing.T) {
		reqBody := CreateUserReq{
			Username: "newuser",
			Password: "password",
			// DisplayName is missing
			IsAdmin: false,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req = adminCtx(req)
		w := httptest.NewRecorder()

		server.createUserHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for missing display_name, got %d", w.Code)
		}
	})

	t.Run("DELETE /api/rooms/{roomID} - Response Body", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/rooms/room1", nil)
		req = adminCtx(req)

		// Add chi context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("roomID", "room1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		server.deleteRoomHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if resp["status"] != "ok" {
			t.Errorf("Expected status 'ok', got %v", resp["status"])
		}
	})

	t.Run("PUT /api/rooms/{roomID} - Response Body", func(t *testing.T) {
		reqBody := map[string]string{"name": "New Name"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/rooms/room1", bytes.NewBuffer(body))
		req = adminCtx(req)

		// Add chi context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("roomID", "room1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		server.renameRoomHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if resp["status"] != "ok" {
			t.Errorf("Expected status 'ok', got %v", resp["status"])
		}
	})

	t.Run("POST /api/change-password - Missing Fields", func(t *testing.T) {
		reqBody := ChangePasswordReq{
			// OldPassword missing
			NewPassword: "newpass",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/change-password", bytes.NewBuffer(body))
		req = userCtx(req)
		w := httptest.NewRecorder()

		server.changePasswordHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("POST /api/login - Bad Request", func(t *testing.T) {
		reqBody := LoginRequest{
			Username: "user",
			// Password missing
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		server.loginHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("POST /api/login - Unauthorized", func(t *testing.T) {
		reqBody := LoginRequest{
			Username: "user",
			Password: "wrongpassword",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		server.loginHandler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
		}
	})
}
