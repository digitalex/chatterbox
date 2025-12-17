import { db, type Room, type Message } from './db';

// 1. DYNAMIC API URL
// Uses the variable from .env.production or .env.local. 
// Fallback to localhost if missing.
export const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

// 2. DYNAMIC USER ID
// Check if we already have an identity in this browser
const STORAGE_KEY = 'chatterbox_user_id';
let storedId = localStorage.getItem(STORAGE_KEY);

if (!storedId) {
  // If not, generate a new random UUID
  storedId = crypto.randomUUID();
  localStorage.setItem(STORAGE_KEY, storedId);
}

// Export the persistent ID (e.g. "a1b2-c3d4-...")
export const USER_ID = storedId!;

// Token Cache
let authToken: string | null = null;

export async function getAuthToken(): Promise<string> {
  if (authToken) return authToken;

  const response = await fetch(`${API_URL}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: USER_ID }),
  });

  if (!response.ok) {
      throw new Error('Login failed');
  }

  const data = await response.json();
  authToken = data.token;
  return authToken!;
}

export async function syncData() {
  try {
    // 1. Get last sync timestamp from local DB
    const config = await db.config.get('last_synced_at');
    const lastSyncedAt = config?.value || null;

    // 2. Gather unsynced items
    const unsyncedRooms = await db.rooms.where('synced').equals(0).toArray();
    const unsyncedMessages = await db.messages.where('synced').equals(0).toArray();

    // 3. Call Server
    const token = await getAuthToken();
    const response = await fetch(`${API_URL}/sync`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify({
        last_synced_at: lastSyncedAt,
        rooms: unsyncedRooms.map(r => ({ room_id: r.room_id, name: r.name })),
        messages: unsyncedMessages.map(m => ({ room_id: m.room_id, message_id: m.message_id, content: m.content }))
      }),
    });

    if (response.status === 401) {
      authToken = null; // Token might be expired
      // Retry logic could be added here
      throw new Error('Unauthorized');
    }

    if (!response.ok) throw new Error('Sync failed');

    const data = await response.json();

    // 4. Write to IndexedDB (Transactional)
    await db.transaction('rw', db.rooms, db.messages, db.config, db.users, async () => {
      
      // A. Mark sent items as synced
      if (unsyncedRooms.length > 0) {
        await Promise.all(unsyncedRooms.map(r => db.rooms.update(r.room_id, { synced: 1 })));
      }
      if (unsyncedMessages.length > 0) {
        // Need composite key for update if not using primary key directly?
        // messages PK is [room_id+message_id]. update() takes the key.
        await Promise.all(unsyncedMessages.map(m => db.messages.update([m.room_id, m.message_id], { synced: 1 })));
      }

      // B. Update Rooms (Downstream)
      if (data.rooms) {
        // We use put to overwrite/update.
        // Important: If we just synced a room we created, server might send it back.
        // We should ensure we don't overwrite local fields if they are newer?
        // But for now, server is truth for other fields.
        // However, we want to keep `synced: 1`.
        // If server sends it back, we can just put it.
        // Map server response to local shape.
        const roomUpdates = data.rooms.map((r: any) => ({
            room_id: r.room_id,
            name: r.name,
            last_read_message_id: r.last_read_message_id,
            // Preserve creation time if we have it, else use now? Server doesn't send CreatedAt in SyncResponse RoomResult?
            // Let's check server RoomResult: { RoomID, Name, LastReadMessageID }. No CreatedAt.
            // If we already have the room, keep created_at. If new, we need it.
            // But wait, our local DB requires created_at.
            // If it's a new room from server (invited), we don't know created_at.
            // We might need to fetch it or just use SyncTimestamp.
            synced: 1
        }));

        for (const r of roomUpdates) {
            const existing = await db.rooms.get(r.room_id);
            await db.rooms.put({
                ...r,
                created_at: existing?.created_at || new Date().toISOString(), // Fallback
                unread_count: existing?.unread_count || 0
            });
        }
      }

      // C. Insert Messages (Downstream)
      if (data.messages) {
        const msgUpdates = data.messages.map((m: any) => ({
            room_id: m.room_id,
            message_id: m.message_id,
            sender_id: m.sender_id,
            content: m.content?.Value || m.content, // Handle Spanner NullJSON structure if raw, but API likely sends unwrapped JSON
            created_at: m.created_at,
            synced: 1
        }));
        await db.messages.bulkPut(msgUpdates);
      }

      // D. Update Users
      if (data.users) {
        await db.users.bulkPut(data.users);
      }

      // E. Update Sync Timestamp
      if (data.sync_timestamp) {
        await db.config.put({ key: 'last_synced_at', value: data.sync_timestamp });
      }
    });

    if ((data.messages?.length || 0) > 0 || unsyncedMessages.length > 0) {
        console.log(`✅ Synced. Sent: ${unsyncedMessages.length}, Recv: ${data.messages?.length || 0}`);
    }
    
  } catch (error) {
    console.error('Sync error:', error);
  }
}

export async function sendMessage(roomId: string, content: any) {
  try {
    // 1. Create local message
    // Use microseconds-ish timestamp to match server int64 expectations if needed,
    // but JS Date.now() is milliseconds. Server previous logic was Microseconds.
    // Let's use Date.now() * 1000 to be safe and compatible with server sorting if it expects micros.
    const messageId = Date.now() * 1000;

    const message: Message = {
        room_id: roomId,
        message_id: messageId,
        sender_id: USER_ID,
        content: content,
        created_at: new Date().toISOString(),
        synced: 0 // Not synced yet
    };

    // 2. Save to DB
    await db.messages.add(message);

    // 3. Trigger Sync
    syncData(); // Fire and forget
    
  } catch (error) {
    console.error('Send error:', error);
  }
}

export async function createRoom(name: string): Promise<Room> {
  try {
    // 1. Create local room
    const roomId = crypto.randomUUID();
    const now = new Date().toISOString();

    const room: Room = {
        room_id: roomId,
        name: name,
        created_at: now,
        last_read_message_id: 0,
        unread_count: 0,
        synced: 0
    };

    // 2. Save to DB
    await db.rooms.add(room);

    // 3. Trigger Sync
    syncData();

    return room;
  } catch (error) {
    console.error('Create room error:', error);
    throw error;
  }
}
