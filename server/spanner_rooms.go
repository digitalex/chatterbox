package main

import (
	"context"

	"cloud.google.com/go/spanner"
)

// DeleteRoom deletes a room and all its related data (due to CASCADE DELETE)
func (db *SpannerDB) DeleteRoom(ctx context.Context, roomID string) error {
	_, err := db.client.Apply(ctx, []*spanner.Mutation{
		spanner.Delete("Rooms", spanner.Key{roomID}),
	})
	return err
}

// RenameRoom updates the name of a room
func (db *SpannerDB) RenameRoom(ctx context.Context, roomID, newName string) error {
	_, err := db.client.Apply(ctx, []*spanner.Mutation{
		spanner.Update("Rooms", []string{"RoomId", "Name"}, []interface{}{roomID, newName}),
	})
	return err
}
