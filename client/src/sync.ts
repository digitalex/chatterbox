import { db } from './db';

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
export const USER_ID = storedId;

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

    // 2. Call Server
    const token = await getAuthToken();
    const response = await fetch(`${API_URL}/sync`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify({ last_synced_at: lastSyncedAt }),
    });

    if (response.status === 401) {
      authToken = null; // Token might be expired
      // Retry logic could be added here
      throw new Error('Unauthorized');
    }

    if (!response.ok) throw new Error('Sync failed');

    const data = await response.json();

    // 3. Write to IndexedDB (Transactional)
    await db.transaction('rw', db.rooms, db.messages, db.config, db.users, async () => {
      
      // A. Update Rooms
      if (data.rooms) {
        await db.rooms.bulkPut(data.rooms);
      }

      // B. Insert Messages
      if (data.messages) {
        await db.messages.bulkPut(data.messages);
      }

      // C. Update Users (New Logic)
      if (data.users) {
        await db.users.bulkPut(data.users);
      }

      // D. Update Sync Timestamp
      if (data.sync_timestamp) {
        await db.config.put({ key: 'last_synced_at', value: data.sync_timestamp });
      }
    });

    if (data.messages?.length > 0) {
        console.log(`✅ Synced. ${data.messages.length} new msgs.`);
    }
    
  } catch (error) {
    console.error('Sync error:', error);
  }
}

export async function sendMessage(roomId: string, content: any) {
  try {
    const token = await getAuthToken();
    const response = await fetch(`${API_URL}/rooms/${roomId}/messages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify({ content }),
    });

    if (response.status === 401) {
        authToken = null;
        throw new Error('Unauthorized');
    }

    if (!response.ok) throw new Error('Send failed');
    
    // Trigger an immediate sync to pull the new message back down
    await syncData();
    
  } catch (error) {
    console.error('Send error:', error);
  }
}

export async function createRoom(name: string) {
  try {
    const token = await getAuthToken();
    const response = await fetch(`${API_URL}/rooms`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify({ name }),
    });

    if (response.status === 401) {
        authToken = null;
        throw new Error('Unauthorized');
    }

    if (!response.ok) throw new Error('Create room failed');

    return await response.json();
  } catch (error) {
    console.error('Create room error:', error);
    throw error;
  }
}
