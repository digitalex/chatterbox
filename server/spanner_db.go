package main

import (
	"context"
	"crypto/rand"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SpannerDB struct {
	client *spanner.Client
}

// Auth methods

func (db *SpannerDB) AuthenticateUser(ctx context.Context, username, password string) (string, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT UserId, PasswordHash, Salt, IsAdmin FROM Users WHERE Username = @username`,
		Params: map[string]interface{}{
			"username": username,
		},
	}
	iter := db.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return "", false, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if err != nil {
		return "", false, err
	}

	var userID string
	var passwordHash []byte
	var salt []byte
	var isAdmin spanner.NullBool

	if err := row.Columns(&userID, &passwordHash, &salt, &isAdmin); err != nil {
		return "", false, err
	}

	// Verify password by appending the stored salt to the provided password
	// and comparing it against the bcrypt hash.
	err = bcrypt.CompareHashAndPassword(passwordHash, []byte(password+string(salt)))
	if err != nil {
		return "", false, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	return userID, isAdmin.Bool, nil
}

func (db *SpannerDB) VerifyPassword(ctx context.Context, userID, password string) error {
	stmt := spanner.Statement{
		SQL: `SELECT PasswordHash, Salt FROM Users WHERE UserId = @uid`,
		Params: map[string]interface{}{
			"uid": userID,
		},
	}
	iter := db.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return err
	}

	var passwordHash []byte
	var salt []byte

	if err := row.Columns(&passwordHash, &salt); err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword(passwordHash, []byte(password+string(salt)))
	if err != nil {
		return status.Error(codes.Unauthenticated, "invalid password")
	}

	return nil
}

func (db *SpannerDB) CreateUser(ctx context.Context, user CreateUserReq) (string, error) {
	userID := uuid.New().String()
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash password + salt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password+string(salt)), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	_, err = db.client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("Users",
			[]string{"UserId", "Username", "PasswordHash", "Salt", "DisplayName", "IsAdmin", "CreatedAt"},
			[]interface{}{userID, user.Username, hashedPassword, salt, user.DisplayName, user.IsAdmin, spanner.CommitTimestamp},
		),
	})
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (db *SpannerDB) UpdatePassword(ctx context.Context, userID, newPassword string) error {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword+string(salt)), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.client.Apply(ctx, []*spanner.Mutation{
		spanner.Update("Users",
			[]string{"UserId", "PasswordHash", "Salt"},
			[]interface{}{userID, hashedPassword, salt},
		),
	})
	return err
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

func (db *SpannerDB) Sync(ctx context.Context, userID string, lastSync time.Time, newRooms []RoomReq, newMessages []MsgReq) ([]*RoomResult, []*MsgResult, []*UserResult, error) {
	// 1. Write new data (Upstream)
	if len(newRooms) > 0 || len(newMessages) > 0 {
		_, err := db.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Check if user exists to ensure FK consistency, fail if not.
			stmt := spanner.Statement{
				SQL:    "SELECT 1 FROM Users WHERE UserId = @uid",
				Params: map[string]interface{}{"uid": userID},
			}
			iter := txn.Query(ctx, stmt)
			_, err := iter.Next()
			iter.Stop()

			if err == iterator.Done {
				return status.Error(codes.NotFound, "user not found")
			} else if err != nil {
				return err
			}

			var mutations []*spanner.Mutation

			// Process new rooms
			for _, r := range newRooms {
				if r.Name == "" {
					return status.Error(codes.InvalidArgument, "room name cannot be empty")
				}
				// Use Insert to prevent overwriting existing rooms
				roomMutation := spanner.Insert("Rooms",
					[]string{"RoomId", "Name", "CreatedAt"},
					[]interface{}{r.RoomID, r.Name, spanner.CommitTimestamp},
				)
				// Insert member. If room already existed (and Insert failed), the transaction will abort.
				// If room is new, this member must also be new for this room.
				memberMutation := spanner.Insert("RoomMembers",
					[]string{"RoomId", "UserId", "JoinedAt", "LastReadMessageId"},
					[]interface{}{r.RoomID, userID, spanner.CommitTimestamp, 0},
				)
				mutations = append(mutations, roomMutation, memberMutation)
			}

			// Process new messages
			for _, m := range newMessages {
				// Use Insert to prevent overwriting messages/timestamps
				msgMutation := spanner.Insert("Messages",
					[]string{"RoomId", "MessageId", "SenderId", "Content", "CreatedAt"},
					[]interface{}{m.RoomID, m.MessageID, userID, spanner.NullJSON{Value: m.Content, Valid: true}, spanner.CommitTimestamp},
				)
				mutations = append(mutations, msgMutation)
			}

			return txn.BufferWrite(mutations)
		})
		if err != nil {
			// If insertion failed (e.g. AlreadyExists), return the error and let the client handle the conflict.
			return nil, nil, nil, err
		}
	} else {
		// Just to be safe, if we are only reading but the user doesn't exist, should we fail?
		// The query `SELECT ... FROM RoomMembers ... WHERE UserId = @uid` will just return empty if user has no rooms.
		// If strict "user must exist" is needed, we could check here too.
		// But usually Sync is safe to run empty. The "write" path is where FK violations happen.
	}

	// 2. Read updates (Downstream)
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
