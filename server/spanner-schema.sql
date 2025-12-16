CREATE TABLE Rooms (
  RoomId STRING(36) NOT NULL,
  Name STRING(255),
  CreatedAt TIMESTAMP NOT NULL OPTIONS (
    allow_commit_timestamp = true
  ),
) PRIMARY KEY(RoomId);

CREATE TABLE Messages (
  RoomId STRING(36) NOT NULL,
  MessageId INT64 NOT NULL,
  SenderId STRING(36) NOT NULL,
  Content JSON,
  CreatedAt TIMESTAMP NOT NULL OPTIONS (
    allow_commit_timestamp = true
  ),
) PRIMARY KEY(RoomId, MessageId DESC),
  INTERLEAVE IN PARENT Rooms ON DELETE CASCADE;

CREATE INDEX MessagesByTime ON Messages(CreatedAt) STORING (SenderId, Content);

CREATE TABLE RoomMembers (
  RoomId STRING(36) NOT NULL,
  UserId STRING(36) NOT NULL,
  JoinedAt TIMESTAMP NOT NULL OPTIONS (
    allow_commit_timestamp = true
  ),
  LastReadMessageId INT64,
) PRIMARY KEY(RoomId, UserId),
  INTERLEAVE IN PARENT Rooms ON DELETE CASCADE;

CREATE TABLE Users (
  UserId STRING(36) NOT NULL,
  Email STRING(255),
  PublicKey STRING(MAX),
  CreatedAt TIMESTAMP NOT NULL OPTIONS (
    allow_commit_timestamp = true
  ),
  DisplayName STRING(100),
) PRIMARY KEY(UserId);

ALTER TABLE RoomMembers ADD CONSTRAINT FK_User FOREIGN KEY(UserId) REFERENCES Users(UserId);
