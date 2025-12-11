import { db } from './db';

//const API_URL = 'http://localhost:8080/api';
export const API_URL = 'https://chatterbox-api-799963617514.us-west1.run.app/api';
export const USER_ID = 'user-alice-123'; // Hardcoded for testing

export async function syncData() {
  try {
    // 1. Get last sync timestamp from local DB
    const config = await db.config.get('last_synced_at');
    const lastSyncedAt = config?.value || null;

    // 2. Call Server
    const response = await fetch(`${API_URL}/sync`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': USER_ID,
      },
      body: JSON.stringify({ last_synced_at: lastSyncedAt }),
    });

    if (!response.ok) throw new Error('Sync failed');

    const data = await response.json();

    // 3. Write to IndexedDB (Transactional)
    await db.transaction('rw', db.rooms, db.messages, db.config, async () => {
      
      // A. Update Rooms
      if (data.rooms) {
        await db.rooms.bulkPut(data.rooms);
      }

      // B. Insert Messages
      if (data.messages) {
        await db.messages.bulkPut(data.messages);
      }

      // C. Update Sync Timestamp
      if (data.sync_timestamp) {
        await db.config.put({ key: 'last_synced_at', value: data.sync_timestamp });
      }
    });

    console.log(`✅ Synced. ${data.messages?.length || 0} new msgs.`);
    
  } catch (error) {
    console.error('Sync error:', error);
  }
}

export async function sendMessage(roomId: string, content: any) {
  try {
    const response = await fetch(`${API_URL}/rooms/${roomId}/messages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': 'user-alice-123', // Still hardcoded for now
      },
      body: JSON.stringify({ content }),
    });

    if (!response.ok) throw new Error('Send failed');
    
    // Trigger an immediate sync to pull the new message back down
    await syncData();
    
  } catch (error) {
    console.error('Send error:', error);
  }
}
