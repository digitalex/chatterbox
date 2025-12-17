package main

import (
	"context"
	"time"
)

type MockDB struct {
	HealthCheckFn    func(ctx context.Context) (int64, error)
	UpdateProfileFn  func(ctx context.Context, userID string, displayName string, publicKey string) error
	GetRoomMembersFn func(ctx context.Context, roomID string) ([]*RoomMember, error)
	SyncFn           func(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error)
}

func (m *MockDB) HealthCheck(ctx context.Context) (int64, error) {
	if m.HealthCheckFn != nil {
		return m.HealthCheckFn(ctx)
	}
	return 1, nil
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

func (m *MockDB) Sync(ctx context.Context, userID string, lastSync time.Time, rooms []RoomReq, messages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
	if m.SyncFn != nil {
		return m.SyncFn(ctx, userID, lastSync, rooms, messages)
	}
	return []*RoomResult{}, []*MsgResult{}, []*UserResult{}, nil
}
