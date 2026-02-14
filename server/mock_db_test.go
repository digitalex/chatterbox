package main

import (
	"context"
	"time"
)

type MockDB struct {
	HealthCheckFn    func(ctx context.Context) (int64, error)
	UpdateProfileFn  func(ctx context.Context, userID string, displayName *string) error
	GetRoomMembersFn func(ctx context.Context, roomID string) ([]*RoomMember, error)
	SyncFn           func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error)
	// Auth
	AuthenticateUserFn func(ctx context.Context, username, password string) (string, bool, error)
	VerifyPasswordFn   func(ctx context.Context, userID, password string) error
	CreateUserFn       func(ctx context.Context, user CreateUserReq) (string, error)
	UpdatePasswordFn   func(ctx context.Context, userID, newPassword string) error

	// Room Management
	DeleteRoomFn func(ctx context.Context, roomID string) error
	RenameRoomFn func(ctx context.Context, roomID, newName string) error
}

func (m *MockDB) HealthCheck(ctx context.Context) (int64, error) {
	if m.HealthCheckFn != nil {
		return m.HealthCheckFn(ctx)
	}
	return 1, nil
}

func (m *MockDB) UpdateProfile(ctx context.Context, userID string, displayName *string) error {
	if m.UpdateProfileFn != nil {
		return m.UpdateProfileFn(ctx, userID, displayName)
	}
	return nil
}

func (m *MockDB) DeleteRoom(ctx context.Context, roomID string) error {
	if m.DeleteRoomFn != nil {
		return m.DeleteRoomFn(ctx, roomID)
	}
	return nil
}

func (m *MockDB) RenameRoom(ctx context.Context, roomID, newName string) error {
	if m.RenameRoomFn != nil {
		return m.RenameRoomFn(ctx, roomID, newName)
	}
	return nil
}

func (m *MockDB) GetRoomMembers(ctx context.Context, roomID string) ([]*RoomMember, error) {
	if m.GetRoomMembersFn != nil {
		return m.GetRoomMembersFn(ctx, roomID)
	}
	return []*RoomMember{}, nil
}

func (m *MockDB) Sync(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
	if m.SyncFn != nil {
		return m.SyncFn(ctx, userID, lastSync, rooms, messages)
	}
	return []*RoomResult{}, []*MsgResult{}, []*UserResult{}, nil
}

func (m *MockDB) AuthenticateUser(ctx context.Context, username, password string) (string, bool, error) {
	if m.AuthenticateUserFn != nil {
		return m.AuthenticateUserFn(ctx, username, password)
	}
	return "mock-user-id", false, nil
}

func (m *MockDB) VerifyPassword(ctx context.Context, userID, password string) error {
	if m.VerifyPasswordFn != nil {
		return m.VerifyPasswordFn(ctx, userID, password)
	}
	return nil
}

func (m *MockDB) CreateUser(ctx context.Context, user CreateUserReq) (string, error) {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, user)
	}
	return "new-user-id", nil
}

func (m *MockDB) UpdatePassword(ctx context.Context, userID, newPassword string) error {
	if m.UpdatePasswordFn != nil {
		return m.UpdatePasswordFn(ctx, userID, newPassword)
	}
	return nil
}
