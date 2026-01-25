package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// TestCreateUser_Compliance verifies POST /api/users
func TestCreateUser_Compliance(t *testing.T) {
	mockDB := new(MockDB)
	server := NewServer(mockDB, nil)

	t.Run("Missing Fields", func(t *testing.T) {
		tests := []struct {
			name    string
			payload string
		}{
			{"Missing Username", `{"password": "pw", "display_name": "dn"}`},
			{"Missing Password", `{"username": "un", "display_name": "dn"}`},
			{"Missing DisplayName", `{"username": "un", "password": "pw"}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("POST", "/api/users", strings.NewReader(tc.payload))
				// Inject admin
				ctx := context.WithValue(req.Context(), userIDKey, "admin")
				ctx = context.WithValue(ctx, isAdminKey, true)
				req = req.WithContext(ctx)

				rr := httptest.NewRecorder()
				server.createUserHandler(rr, req)

				assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected 400 for %s", tc.name)
			})
		}
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		mockDB.CreateUserFn = func(ctx context.Context, user CreateUserReq) (string, error) {
			return "", fmt.Errorf("db error")
		}

		payload := `{"username": "un", "password": "pw", "display_name": "dn"}`
		req, _ := http.NewRequest("POST", "/api/users", strings.NewReader(payload))
		// Inject admin
		ctx := context.WithValue(req.Context(), userIDKey, "admin")
		ctx = context.WithValue(ctx, isAdminKey, true)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		server.createUserHandler(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

// TestLogin_Compliance verifies POST /api/login
func TestLogin_Compliance(t *testing.T) {
	mockDB := new(MockDB)
	server := NewServer(mockDB, nil)

	t.Run("Missing Credentials", func(t *testing.T) {
		tests := []struct {
			name    string
			payload string
		}{
			{"Missing Username", `{"password": "pw"}`},
			{"Missing Password", `{"username": "un"}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("POST", "/api/login", strings.NewReader(tc.payload))
				rr := httptest.NewRecorder()
				server.loginHandler(rr, req)
				assert.Equal(t, http.StatusBadRequest, rr.Code)
			})
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockDB.AuthenticateUserFn = func(ctx context.Context, u, p string) (string, bool, error) {
			return "", false, fmt.Errorf("auth error")
		}

		payload := `{"username": "un", "password": "pw"}`
		req, _ := http.NewRequest("POST", "/api/login", strings.NewReader(payload))
		rr := httptest.NewRecorder()
		server.loginHandler(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

// TestChangePassword_Compliance verifies POST /api/change-password
func TestChangePassword_Compliance(t *testing.T) {
	mockDB := new(MockDB)
	server := NewServer(mockDB, nil)

	t.Run("Missing Fields", func(t *testing.T) {
		tests := []struct {
			name    string
			payload string
		}{
			{"Missing Old Password", `{"new_password": "np"}`},
			{"Missing New Password", `{"old_password": "op"}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("POST", "/api/change-password", strings.NewReader(tc.payload))
				// Inject user
				ctx := context.WithValue(req.Context(), userIDKey, "user")
				req = req.WithContext(ctx)

				rr := httptest.NewRecorder()
				server.changePasswordHandler(rr, req)
				assert.Equal(t, http.StatusBadRequest, rr.Code)
			})
		}
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		mockDB.VerifyPasswordFn = func(ctx context.Context, u, p string) error { return nil }
		mockDB.UpdatePasswordFn = func(ctx context.Context, u, p string) error {
			return fmt.Errorf("db error")
		}

		payload := `{"old_password": "op", "new_password": "np"}`
		req, _ := http.NewRequest("POST", "/api/change-password", strings.NewReader(payload))
		ctx := context.WithValue(req.Context(), userIDKey, "user")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		server.changePasswordHandler(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

// TestRoomManagement_Compliance verifies DELETE/PUT rooms
func TestRoomManagement_Compliance(t *testing.T) {
	mockDB := new(MockDB)
	server := NewServer(mockDB, nil)

	t.Run("Delete Room - Success Response", func(t *testing.T) {
		mockDB.DeleteRoomFn = func(ctx context.Context, id string) error { return nil }
		req, _ := http.NewRequest("DELETE", "/api/rooms/r1", nil)
		ctx := context.WithValue(req.Context(), userIDKey, "admin")
		ctx = context.WithValue(ctx, isAdminKey, true)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("roomID", "r1")
		req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		server.deleteRoomHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, `{"status": "ok"}`, rr.Body.String())
	})

	t.Run("Delete Room - 500", func(t *testing.T) {
		mockDB.DeleteRoomFn = func(ctx context.Context, id string) error { return fmt.Errorf("db error") }
		req, _ := http.NewRequest("DELETE", "/api/rooms/r1", nil)
		ctx := context.WithValue(req.Context(), userIDKey, "admin")
		ctx = context.WithValue(ctx, isAdminKey, true)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("roomID", "r1")
		req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		server.deleteRoomHandler(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("Rename Room - Success Response", func(t *testing.T) {
		mockDB.RenameRoomFn = func(ctx context.Context, id, name string) error { return nil }
		req, _ := http.NewRequest("PUT", "/api/rooms/r1", bytes.NewBufferString(`{"name":"New Name"}`))
		ctx := context.WithValue(req.Context(), userIDKey, "admin")
		ctx = context.WithValue(ctx, isAdminKey, true)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("roomID", "r1")
		req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		server.renameRoomHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, `{"status": "ok"}`, rr.Body.String())
	})

	t.Run("Rename Room - 500", func(t *testing.T) {
		mockDB.RenameRoomFn = func(ctx context.Context, id, name string) error { return fmt.Errorf("db error") }
		req, _ := http.NewRequest("PUT", "/api/rooms/r1", bytes.NewBufferString(`{"name":"New Name"}`))
		ctx := context.WithValue(req.Context(), userIDKey, "admin")
		ctx = context.WithValue(ctx, isAdminKey, true)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("roomID", "r1")
		req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		server.renameRoomHandler(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
