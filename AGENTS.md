# AGENTS.md

## Project Overview

**Chatterbox** is a local-first, family-oriented chat application. It prioritizes secure authentication and offline-first capabilities.

### Core Philosophy

1. **Local-First:** The client (React + Dexie) is the source of truth for the UI. Users write to IndexedDB immediately.
2. **Sync Loop:** A background process synchronizes local IndexedDB data with the remote Spanner database.
3. **User-Based Auth:** Access is granted via Username/Password login, which returns a JWT. User management is handled by administrators.

---

## Tech Stack

### Frontend

* **Framework:** React 19+ (TypeScript), Vite.
* **State/Storage:** `dexie` (IndexedDB wrapper) and `dexie-react-hooks`.
* **Styling:** Custom CSS (`App.css`). No Tailwind/Bootstrap. Light mode is default.
* **Routing:** Custom state-based routing (Room ID tracking).

### Backend

* **Language:** Go (Golang).
* **Database:** Google Cloud Spanner.
* **Router:** `chi`.

---

## Database Architecture

### 1. Cloud Spanner (Server Source of Truth)

We utilize **Interleaving** to optimize for locality. Messages and Members are stored physically close to their parent Room.

See `server/spanner-schema.sql` for the most up to date schema definition.

---

### 2. Dexie.js (Client Local Store)

The client mirrors the server schema but adds synchronization flags.

* **`rooms`**: `room_id, name, created_at, synced (0|1)`
* **`messages`**: `[room_id+message_id], room_id, content, created_at, synced (0|1)` (Compound PK)
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

Used for **Authentication & Profile Management**.

* These **CANNOT** be local-first because they require server authority (e.g., password validation).
* **Login Flow:**
1. `POST /api/login` with `{ username, password }`.
2. Server validates credentials against `Users` table.
3. Server returns JWT Token.
4. Client stores token and triggers **Sync Loop** to fetch user data and rooms.

* **Other Endpoints:** `POST /api/me` (Update Profile), `POST /api/change-password`, `POST /api/users` (Admin Create).


---

## Coding style

* Comments - use sparingly. Comments should never be about the "what", but it can sometimes be helpful to explain the "why". In general aim for clear, readable code over comments.
* Nesting - limit nesting by using early returns and factoring out helper methods.
* Organization - aim for separation of concerns; high cohesion, low coupling.
* State:
  - Prefer immutable objects when practical. They are generally easier to reason about and test than mutable, stateful code.
  - Any object or database table holding state should be designed to make illegal states impossible and __unrepresentable__.

---

## Common Tasks & Rules for Agents

1. **Adding UI Features:** Always implement the `useLiveQuery` hook from Dexie. Never fetch from the API directly in a component (except for RPCs like Login).
2. **Schema Changes:** If you modify the DB, you must update:
* The Spanner DDL (Server).
* The Dexie Schema definition (`db.ts`).
* The Sync Logic (`sync.ts`) to handle the new field.


3. **Security:** Never trust the client's `SenderId` in a payload. Always override it with the JWT `UserId` on the server.
4. **Formatting:** Use TypeScript for all client code. Use Go idioms (context, error handling) for backend.
