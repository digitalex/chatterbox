package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/spanner"
)

func main() {
	ctx := context.Background()
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	instanceID := "chatterbox-db"
	databaseID := os.Getenv("SPANNER_DATABASE")

	if projectID == "" || databaseID == "" {
		log.Fatal("Please set GOOGLE_CLOUD_PROJECT and SPANNER_DATABASE")
	}

	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)
	client, err := spanner.NewClient(ctx, dbPath)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 1. Data to Insert
	userID := "user-alice-123"
	roomID := "room-general-001"
	
	// Create mutations (using InsertOrUpdate to make it idempotent - safe to run twice)
	mutations := []*spanner.Mutation{
		// A. User
		spanner.InsertOrUpdate("Users",
			[]string{"UserId", "Email", "PublicKey", "CreatedAt"},
			[]interface{}{userID, "alice@example.com", "dummy-public-key-base64", spanner.CommitTimestamp},
		),
		
		// B. Room
		spanner.InsertOrUpdate("Rooms",
			[]string{"RoomId", "Name", "CreatedAt"},
			[]interface{}{roomID, "General Chat", spanner.CommitTimestamp},
		),

		// C. Membership
		spanner.InsertOrUpdate("RoomMembers",
			[]string{"RoomId", "UserId", "JoinedAt", "LastReadMessageId"},
			[]interface{}{roomID, userID, spanner.CommitTimestamp, 0},
		),

		// D. Messages (Note the JSON content)
		spanner.InsertOrUpdate("Messages",
			[]string{"RoomId", "MessageId", "SenderId", "Content", "CreatedAt"},
			[]interface{}{roomID, 1001, userID, spanner.NullJSON{Value: map[string]interface{}{"text": "Hello World!"}, Valid: true}, spanner.CommitTimestamp},
		),
		spanner.InsertOrUpdate("Messages",
			[]string{"RoomId", "MessageId", "SenderId", "Content", "CreatedAt"},
			[]interface{}{roomID, 1002, userID, spanner.NullJSON{Value: map[string]interface{}{"text": "Is anyone here?"}, Valid: true}, spanner.CommitTimestamp},
		),
	}

	// 2. Apply Mutations
	_, err = client.Apply(ctx, mutations)
	if err != nil {
		log.Fatalf("Failed to seed data: %v", err)
	}

	fmt.Println("✅ Seed data inserted successfully!")
	fmt.Printf("User: %s\nRoom: %s\n", userID, roomID)
}

