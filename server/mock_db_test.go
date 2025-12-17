package main

import (
	"context"
	"time"
)

type MockDB struct {
	HealthCheckFn    func(ctx context.Context) (int64, error)
	CreateRoomFn     func(ctx context.Context, roomID string, name string, userID string) error
	UpdateProfileFn  func(ctx context.Context, userID string, displayName string, publicKey string) error
	GetRoomMembersFn func(ctx context.Context, roomID string) ([]*RoomMember, error)
	SendMessageFn    func(ctx context.Context, roomID string, userID string, msgID int64, content interface{}) error
	SyncFn           func(ctx context.Context, userID string, lastSync time.Time) ([]*RoomResult, []*MsgResult, []*UserResult, error)
    IsRoomOwnerFn    func(ctx context.Context, roomID string, userID string) (bool, error)
    GenerateInviteFn func(ctx context.Context, roomID string, inviteCode string, createdBy string, expiresAt time.Time) error
    AcceptInviteFn   func(ctx context.Context, inviteCode string, userID string) (string, error)
}

func (m *MockDB) HealthCheck(ctx context.Context) (int64, error) {
	if m.HealthCheckFn != nil {
		return m.HealthCheckFn(ctx)
	}
	return 1, nil
}

func (m *MockDB) CreateRoom(ctx context.Context, roomID string, name string, userID string) error {
	if m.CreateRoomFn != nil {
		return m.CreateRoomFn(ctx, roomID, name, userID)
	}
	return nil
}

func (m *MockDB) UpdateProfile(ctx context.Context, userID string, displayName string, publicKey string) error {
	if m.UpdateProfileFn != nil {
		return m.UpdateProfileFn(ctx, userID, displayName, publicKey)
	}
	return nil
}

func (m *MockDB) GetRoomMembers(ctx context.Context, roomID string) ([]*RoomMember, error) {
	if m.GetRoomMembersFn != nil {
		return m.GetRoomMembersFn(ctx, roomID)
	}
	return []*RoomMember{}, nil
}

func (m *MockDB) SendMessage(ctx context.Context, roomID string, userID string, msgID int64, content interface{}) error {
	if m.SendMessageFn != nil {
		return m.SendMessageFn(ctx, roomID, userID, msgID, content)
	}
	return nil
}

func (m *MockDB) Sync(ctx context.Context, userID string, lastSync time.Time) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
	if m.SyncFn != nil {
		return m.SyncFn(ctx, userID, lastSync)
	}
	return []*RoomResult{}, []*MsgResult{}, []*UserResult{}, nil
}

func (m *MockDB) IsRoomOwner(ctx context.Context, roomID string, userID string) (bool, error) {
    if m.IsRoomOwnerFn != nil {
        return m.IsRoomOwnerFn(ctx, roomID, userID)
    }
    return false, nil
}

func (m *MockDB) GenerateInvite(ctx context.Context, roomID string, inviteCode string, createdBy string, expiresAt time.Time) error {
    if m.GenerateInviteFn != nil {
        return m.GenerateInviteFn(ctx, roomID, inviteCode, createdBy, expiresAt)
    }
    return nil
}

func (m *MockDB) AcceptInvite(ctx context.Context, inviteCode string, userID string) (string, error) {
    if m.AcceptInviteFn != nil {
        return m.AcceptInviteFn(ctx, inviteCode, userID)
    }
    return "", nil
}
