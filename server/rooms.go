package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) deleteRoomHandler(w http.ResponseWriter, r *http.Request) {
	if !IsAdminFromContext(r.Context()) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	roomID := chi.URLParam(r, "roomID")
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteRoom(r.Context(), roomID); err != nil {
		http.Error(w, "Failed to delete room", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) renameRoomHandler(w http.ResponseWriter, r *http.Request) {
	if !IsAdminFromContext(r.Context()) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	roomID := chi.URLParam(r, "roomID")
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	if err := s.db.RenameRoom(r.Context(), roomID, req.Name); err != nil {
		http.Error(w, "Failed to rename room", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
