-- 1. Users Table
CREATE TABLE Users (
    UserId STRING(36) NOT NULL,
    Email STRING(255) NOT NULL,
    PublicKey STRING(MAX),
    CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (UserId);

-- 2. Rooms Table
CREATE TABLE Rooms (
    RoomId STRING(36) NOT NULL,
    Name STRING(255),
    CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RoomId);

-- 3. RoomMembers (Interleaved in Rooms)
CREATE TABLE RoomMembers (
    RoomId STRING(36) NOT NULL,
    UserId STRING(36) NOT NULL,
    JoinedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    LastReadMessageId INT64,
    CONSTRAINT FK_User FOREIGN KEY (UserId) REFERENCES Users (UserId)
) PRIMARY KEY (RoomId, UserId),
  INTERLEAVE IN PARENT Rooms ON DELETE CASCADE;

-- 4. Messages (Interleaved in Rooms)
CREATE TABLE Messages (
    RoomId STRING(36) NOT NULL,
    MessageId INT64 NOT NULL,
    SenderId STRING(36) NOT NULL,
    Content JSON, -- Supports E2EE payload structure
    CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RoomId, MessageId DESC),
  INTERLEAVE IN PARENT Rooms ON DELETE CASCADE;

-- 5. Global Index for Sync
CREATE INDEX MessagesByTime ON Messages(CreatedAt) STORING (SenderId, Content);

