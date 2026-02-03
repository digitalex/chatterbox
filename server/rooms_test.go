package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestDeleteRoomHandler(t *testing.T) {
	mockDB := new(MockDB)
	server := NewServer(mockDB, nil)

	t.Run("Success", func(t *testing.T) {
		roomID := "room123"

		mockDB.DeleteRoomFn = func(ctx context.Context, id string) error {
			assert.Equal(t, roomID, id)
			return nil
		}

		req, _ := http.NewRequest("DELETE", "/api/rooms/"+roomID, nil)
		// Inject admin user
		ctx := context.WithValue(req.Context(), userIDKey, "admin-user")
		ctx = context.WithValue(ctx, isAdminKey, true)
		req = req.WithContext(ctx)

		// Create chi context with URL param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("roomID", roomID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		server.deleteRoomHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, `{"status":"ok"}`, rr.Body.String())
	})

	t.Run("Forbidden", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/rooms/room123", nil)
		// Inject non-admin user
		ctx := context.WithValue(req.Context(), userIDKey, "normal-user")
		ctx = context.WithValue(ctx, isAdminKey, false)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		server.deleteRoomHandler(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}

func TestRenameRoomHandler(t *testing.T) {
	mockDB := new(MockDB)
	server := NewServer(mockDB, nil)

	t.Run("Success", func(t *testing.T) {
		roomID := "room123"
		newName := "New Name"

		mockDB.RenameRoomFn = func(ctx context.Context, id, name string) error {
			assert.Equal(t, roomID, id)
			assert.Equal(t, newName, name)
			return nil
		}

		body, _ := json.Marshal(map[string]string{"name": newName})
		req, _ := http.NewRequest("PUT", "/api/rooms/"+roomID, bytes.NewBuffer(body))

		// Inject admin user
		ctx := context.WithValue(req.Context(), userIDKey, "admin-user")
		ctx = context.WithValue(ctx, isAdminKey, true)
		req = req.WithContext(ctx)

		// Create chi context with URL param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("roomID", roomID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		server.renameRoomHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, `{"status":"ok"}`, rr.Body.String())
	})

	t.Run("Forbidden", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/rooms/room123", nil)
		// Inject non-admin user
		ctx := context.WithValue(req.Context(), userIDKey, "normal-user")
		ctx = context.WithValue(ctx, isAdminKey, false)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		server.renameRoomHandler(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("Missing Name", func(t *testing.T) {
		roomID := "room123"
		body, _ := json.Marshal(map[string]string{"name": ""})
		req, _ := http.NewRequest("PUT", "/api/rooms/"+roomID, bytes.NewBuffer(body))

		// Inject admin user
		ctx := context.WithValue(req.Context(), userIDKey, "admin-user")
		ctx = context.WithValue(ctx, isAdminKey, true)
		req = req.WithContext(ctx)

		// Create chi context with URL param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("roomID", roomID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		server.renameRoomHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
