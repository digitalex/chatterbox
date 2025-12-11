import { db } from './db';
import { decryptMessage, encryptMessage, importPublicKey } from './crypto';

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

      // // B. Insert Messages
      // if (data.messages) {
      //   await db.messages.bulkPut(data.messages);
      // }

      // Decrypt and store messages
      const decryptedMessages = await Promise.all(data.messages.map(async (msg: any) => {
          // Check if it's an encrypted message
          if (msg.content && msg.content.type === 'e2ee') {
              try {
                  // Get my private key
                  const privKey = (await db.config.get('private_key'))?.value;
                  if (!privKey) return { ...msg, content: { text: "🔑 No Private Key found" } };

                  // Find the key meant for ME
                  const myEncryptedKey = msg.content.keys[USER_ID];
                  if (!myEncryptedKey) return { ...msg, content: { text: "🚫 Not encrypted for me" } };

                  // Decrypt
                  const plainText = await decryptMessage(
                      msg.content.ciphertext,
                      msg.content.iv,
                      myEncryptedKey,
                      privKey
                  );
                  
                  // Replace content with readable text
                  return { ...msg, content: { text: plainText } };
              } catch (e) {
                  return { ...msg, content: { text: "💥 Decryption Error" } };
              }
          }
          // Return legacy/plain messages as-is
          return msg;
      }));

      // NOW save to DB
      if (decryptedMessages.length > 0) {
          await db.messages.bulkPut(decryptedMessages);
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
    // 1. Get Room Members (Server-side endpoint we created)
    const membersRes = await fetch(`${API_URL}/rooms/${roomId}/members`);
    const members = await membersRes.json(); // [{ user_id, public_key }, ...]

    // 2. Prepare Key Map
    const recipientKeys: Record<string, CryptoKey> = {};
    
    for (const m of members) {
        if (m.public_key) {
            recipientKeys[m.user_id] = await importPublicKey(m.public_key);
        }
    }

    // 3. ENCRYPT (Now using the helper!)
    // We send plain text in, we get the complex E2EE JSON out.
    const e2eePayload = await encryptMessage(content.text, recipientKeys);

    // 4. Send to Server
    await fetch(`${API_URL}/rooms/${roomId}/messages`, {
        method: 'POST',
        headers: { 
            'Content-Type': 'application/json', 
            'X-User-ID': USER_ID 
        },
        // Wrap it in "content" to match our Spanner JSON structure
        body: JSON.stringify({ content: e2eePayload }),
    });

    // 5. Sync immediately to see our own message
    await syncData();
    
  } catch (error) {
    console.error('Send error:', error);
  }
}
