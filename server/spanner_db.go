package main

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type SpannerDB struct {
	client *spanner.Client
}

func (db *SpannerDB) HealthCheck(ctx context.Context) (int64, error) {
	stmt := spanner.Statement{SQL: "SELECT 1"}
	iter := db.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return 0, nil // Or error?
	}
	if err != nil {
		return 0, err
	}

	var val int64
	if err := row.Column(0, &val); err != nil {
		return 0, err
	}
	return val, nil
}

func (db *SpannerDB) CreateRoom(ctx context.Context, roomID string, name string, userID string) error {
	roomMutation := spanner.Insert("Rooms",
		[]string{"RoomId", "Name", "CreatedAt", "OwnerId"},
		[]interface{}{roomID, name, spanner.CommitTimestamp, userID},
	)

	memberMutation := spanner.Insert("RoomMembers",
		[]string{"RoomId", "UserId", "JoinedAt", "LastReadMessageId"},
		[]interface{}{roomID, userID, spanner.CommitTimestamp, 0},
	)

	_, err := db.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    "SELECT 1 FROM Users WHERE UserId = @uid",
			Params: map[string]interface{}{"uid": userID},
		}
		iter := txn.Query(ctx, stmt)
		_, err := iter.Next()
		iter.Stop()

		var mutations []*spanner.Mutation

		if err == iterator.Done {
			userMutation := spanner.Insert("Users",
				[]string{"UserId", "DisplayName", "Email", "PublicKey", "CreatedAt"},
				[]interface{}{userID, "Anonymous", "anon", "", spanner.CommitTimestamp},
			)
			mutations = append(mutations, userMutation)
		} else if err != nil {
			return err
		}

		mutations = append(mutations, roomMutation, memberMutation)
		return txn.BufferWrite(mutations)
	})

	return err
}

func (db *SpannerDB) UpdateProfile(ctx context.Context, userID string, displayName string, publicKey string) error {
	userMutation := spanner.InsertOrUpdate("Users",
		[]string{"UserId", "DisplayName", "Email", "PublicKey", "CreatedAt"},
		[]interface{}{userID, displayName, "anon", publicKey, spanner.CommitTimestamp},
	)

	roomMutation := spanner.InsertOrUpdate("RoomMembers",
		[]string{"RoomId", "UserId", "JoinedAt", "LastReadMessageId"},
		[]interface{}{"room-general-001", userID, spanner.CommitTimestamp, 0},
	)

	_, err := db.client.Apply(ctx, []*spanner.Mutation{userMutation, roomMutation})
	return err
}

func (db *SpannerDB) GetRoomMembers(ctx context.Context, roomID string) ([]*RoomMember, error) {
	stmt := spanner.Statement{
		SQL: `SELECT u.UserId, u.PublicKey
              FROM RoomMembers rm
              JOIN Users u ON rm.UserId = u.UserId
              WHERE rm.RoomId = @rid`,
		Params: map[string]interface{}{"rid": roomID},
	}
	iter := db.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var members []*RoomMember
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var m RoomMember
		var pk spanner.NullString
		if err := row.Columns(&m.UserID, &pk); err != nil {
			return nil, err
		}
		m.PublicKey = pk.StringVal
		members = append(members, &m)
	}
	return members, nil
}

func (db *SpannerDB) SendMessage(ctx context.Context, roomID string, userID string, msgID int64, content interface{}) error {
	m := spanner.Insert("Messages",
		[]string{"RoomId", "MessageId", "SenderId", "Content", "CreatedAt"},
		[]interface{}{roomID, msgID, userID, spanner.NullJSON{Value: content, Valid: true}, spanner.CommitTimestamp},
	)

	_, err := db.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

func (db *SpannerDB) Sync(ctx context.Context, userID string, lastSync time.Time) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
	roomIter := db.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT r.RoomId, r.Name, rm.LastReadMessageId
              FROM RoomMembers rm
              JOIN Rooms r ON rm.RoomId = r.RoomId
              WHERE rm.UserId = @uid`,
		Params: map[string]interface{}{"uid": userID},
	})
	defer roomIter.Stop()

	var rooms []*RoomResult
	var myRoomIDs []string

	for {
		row, err := roomIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, nil, err
		}
		var r RoomResult
		var lastRead spanner.NullInt64
		if err := row.Columns(&r.RoomID, &r.Name, &lastRead); err != nil {
			return nil, nil, nil, err
		}
		r.LastReadMessageID = lastRead.Int64
		rooms = append(rooms, &r)
		myRoomIDs = append(myRoomIDs, r.RoomID)
	}

	var messages []*MsgResult
	if len(myRoomIDs) > 0 {
		stmt := spanner.Statement{
			SQL: `SELECT RoomId, MessageId, SenderId, Content, CreatedAt
			      FROM Messages@{FORCE_INDEX=MessagesByTime}
			      WHERE CreatedAt > @since
			      AND RoomId IN UNNEST(@rooms)
			      ORDER BY CreatedAt ASC`,
			Params: map[string]interface{}{
				"since": lastSync,
				"rooms": myRoomIDs,
			},
		}
		msgIter := db.client.Single().Query(ctx, stmt)
		defer msgIter.Stop()

		for {
			row, err := msgIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, nil, nil, err
			}
			var m MsgResult
			if err := row.Columns(&m.RoomID, &m.MessageID, &m.SenderID, &m.Content, &m.CreatedAt); err != nil {
				return nil, nil, nil, err
			}
			messages = append(messages, &m)
		}
	}

	var users []*UserResult
	if len(messages) > 0 {
		uniqueUsers := make(map[string]bool)
		var userIDs []string
		for _, m := range messages {
			if !uniqueUsers[m.SenderID] {
				uniqueUsers[m.SenderID] = true
				userIDs = append(userIDs, m.SenderID)
			}
		}

		stmt := spanner.Statement{
			SQL: `SELECT UserId, DisplayName
                  FROM Users
                  WHERE UserId IN UNNEST(@uids)`,
			Params: map[string]interface{}{"uids": userIDs},
		}
		userIter := db.client.Single().Query(ctx, stmt)
		defer userIter.Stop()

		for {
			row, err := userIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, nil, nil, err
			}
			var u UserResult
			if err := row.Columns(&u.UserID, &u.DisplayName); err != nil {
				continue
			}
			users = append(users, &u)
		}
	}

	return rooms, messages, users, nil
}

// IsRoomOwner checks if the user is the owner of the room
func (db *SpannerDB) IsRoomOwner(ctx context.Context, roomID string, userID string) (bool, error) {
    stmt := spanner.Statement{
        SQL: "SELECT OwnerId FROM Rooms WHERE RoomId = @roomID",
        Params: map[string]interface{}{"roomID": roomID},
    }
    iter := db.client.Single().Query(ctx, stmt)
    defer iter.Stop()

    row, err := iter.Next()
    if err == iterator.Done {
        return false, nil // Room not found
    }
    if err != nil {
        return false, err
    }

    var ownerID spanner.NullString
    if err := row.Column(0, &ownerID); err != nil {
        return false, err
    }

    return ownerID.Valid && ownerID.StringVal == userID, nil
}

// GenerateInvite creates a new invite code for a room
func (db *SpannerDB) GenerateInvite(ctx context.Context, roomID string, inviteCode string, createdBy string, expiresAt time.Time) error {
    m := spanner.Insert("Invites",
        []string{"RoomId", "InviteCode", "CreatedBy", "ExpiresAt", "IsUsed"},
        []interface{}{roomID, inviteCode, createdBy, expiresAt, false},
    )
    _, err := db.client.Apply(ctx, []*spanner.Mutation{m})
    return err
}

// AcceptInvite uses an invite code to join a room
func (db *SpannerDB) AcceptInvite(ctx context.Context, inviteCode string, userID string) (string, error) {
    var roomID string

    // Using ReadWriteTransaction to ensure atomicity
    _, err := db.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        // 1. Find the invite
        stmt := spanner.Statement{
            SQL: `SELECT RoomId, ExpiresAt, IsUsed
                  FROM Invites
                  WHERE InviteCode = @code`,
            Params: map[string]interface{}{"code": inviteCode},
        }
        iter := txn.Query(ctx, stmt)
        defer iter.Stop()

        row, err := iter.Next()
        if err == iterator.Done {
            return status.Errorf(codes.NotFound, "invite not found")
        }
        if err != nil {
            return err
        }

        var expiresAt time.Time
        var isUsed bool
        if err := row.Columns(&roomID, &expiresAt, &isUsed); err != nil {
            return err
        }

        // 2. Validation
        if isUsed {
            return status.Errorf(codes.FailedPrecondition, "invite already used")
        }
        if time.Now().After(expiresAt) {
            return status.Errorf(codes.FailedPrecondition, "invite expired")
        }

        // 3. Mark invite as used
        inviteMutation := spanner.Update("Invites",
            []string{"RoomId", "InviteCode", "IsUsed"},
            []interface{}{roomID, inviteCode, true},
        )

        // 4. Add user to room
        memberMutation := spanner.InsertOrUpdate("RoomMembers",
            []string{"RoomId", "UserId", "JoinedAt", "LastReadMessageId"},
            []interface{}{roomID, userID, spanner.CommitTimestamp, 0},
        )

        // Ensure user exists
        userCheckStmt := spanner.Statement{
			SQL:    "SELECT 1 FROM Users WHERE UserId = @uid",
			Params: map[string]interface{}{"uid": userID},
		}
		userIter := txn.Query(ctx, userCheckStmt)
		_, userErr := userIter.Next()
		userIter.Stop()

		var mutations []*spanner.Mutation

		if userErr == iterator.Done {
			userMutation := spanner.Insert("Users",
				[]string{"UserId", "DisplayName", "Email", "PublicKey", "CreatedAt"},
				[]interface{}{userID, "Anonymous", "anon", "", spanner.CommitTimestamp},
			)
			mutations = append(mutations, userMutation)
		} else if userErr != nil {
			return userErr
		}

        mutations = append(mutations, inviteMutation, memberMutation)
        return txn.BufferWrite(mutations)
    })

    if err != nil {
        return "", err
    }
    return roomID, nil
}
