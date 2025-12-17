# AGENTS.md

## Project Overview

**Chatterbox** is a local-first, family-oriented chat application. It prioritizes low-friction onboarding (no email/passwords) and offline-first capabilities.

### Core Philosophy

1. **Local-First:** The client (React + Dexie) is the source of truth for the UI. Users write to IndexedDB immediately.
2. **Sync Loop:** A background process synchronizes local IndexedDB data with the remote Spanner database.
3. **Possession-Based Auth:** Access is granted via Invite Codes, not user accounts. If a user has the code, they are authenticated.

---

## Tech Stack

### Frontend

* **Framework:** React 18+ (TypeScript), Vite.
* **State/Storage:** `dexie` (IndexedDB wrapper) and `dexie-react-hooks`.
* **Styling:** Custom CSS (`App.css`). No Tailwind/Bootstrap. Dark mode is default.
* **Routing:** Custom state-based routing (Room ID tracking).

### Backend

* **Language:** Go (Golang).
* **Database:** Google Cloud Spanner.
* **Router:** `chi` or standard `net/http`.

---

## Database Architecture

### 1. Cloud Spanner (Server Source of Truth)

We utilize **Interleaving** to optimize for locality. Messages and Members are stored physically close to their parent Room.

See `server/spanner-schema.sql` for the most up to date schema definition.

---

### 2. Dexie.js (Client Local Store)

The client mirrors the server schema but adds synchronization flags.

* **`rooms`**: `room_id, name, created_at, synced (0|1)`
* **`messages`**: `message_id, room_id, content, created_at, synced (0|1)`
* **`users`**: `user_id, display_name` (Read-only cache of directory)

---

## Data Flow & Synchronization

### Pattern A: The Sync Loop (Background)

Used for **Content Creation** (Messages, New Rooms).

1. **User Action:** User sends message -> Saved to Dexie (`synced: 0`).
2. **Sync Process:**
* Queries `synced: 0` records.
* `POST /api/sync` -> Server validates and inserts.
* On success, Client updates `synced: 1`.
* Server responds with *new* data from other users (delta sync).



### Pattern B: RPC / Standalone Calls (Immediate)

Used for **Permissions & Discovery** (Joining, Inviting).

* These **CANNOT** be local-first because they require server authority (e.g., uniqueness of a short code).
* **Join Flow:**
1. `POST /api/join` with `{ inviteCode }`.
2. Server validates code via Spanner Transaction.
3. Server adds user to `RoomMembers`.
4. Client receives `RoomId` + `AuthToken`.
5. Client immediately triggers **Sync Loop** to fetch the room content.



---

## Authorization Model

* **Authentication:** JWT containing `sub` (UserId) and `name`.
* **Room Access:** Validated by checking the `RoomMembers` table.
* **Invite Logic:**
* Only `Room.OwnerId` can generate invites.
* Invites are one-time use and expire (checked via Spanner Transaction).



## Common Tasks & Rules for Agents

1. **Adding UI Features:** Always implement the `useLiveQuery` hook from Dexie. Never fetch from the API directly in a component (except for RPCs like Join).
2. **Schema Changes:** If you modify the DB, you must update:
* The Spanner DDL (Server).
* The Dexie Schema definition (`db.ts`).
* The Sync Logic (`sync.ts`) to handle the new field.


3. **Security:** Never trust the client's `SenderId` in a payload. Always override it with the JWT `UserId` on the server.
4. **Formatting:** Use TypeScript for all client code. Use Go idioms (context, error handling) for backend.
