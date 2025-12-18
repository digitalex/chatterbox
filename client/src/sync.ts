import { db, type Room, type Message } from './db';

// 1. DYNAMIC API URL
// Uses the variable from .env.production or .env.local. 
// Fallback to localhost if missing.
export const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

// 2. DYNAMIC USER ID
const USER_ID_KEY = 'chatterbox_user_id';
const TOKEN_KEY = 'chatterbox_auth_token';

export let USER_ID = localStorage.getItem(USER_ID_KEY);
let authToken = localStorage.getItem(TOKEN_KEY);

export function setAuthInfo(userId: string, token: string) {
  USER_ID = userId;
  authToken = token;
  localStorage.setItem(USER_ID_KEY, userId);
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearAuthInfo() {
  USER_ID = null;
  authToken = null;
  localStorage.removeItem(USER_ID_KEY);
  localStorage.removeItem(TOKEN_KEY);
}

export async function getAuthToken(): Promise<string> {
  if (authToken) return authToken;
  throw new Error('Not logged in');
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
      // Token might be expired
      // We could clearAuthInfo() here or let the UI handle it.
      // For now just throw.
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
        const roomUpdates = data.rooms.map((r: any) => ({
            room_id: r.room_id,
            name: r.name,
            last_read_message_id: r.last_read_message_id,
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
  if (!USER_ID) throw new Error("User not logged in");
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
