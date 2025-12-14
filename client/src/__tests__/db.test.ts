import { describe, it, expect, beforeEach } from 'vitest';
import { db } from '../db';

describe('Database', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  it('should store and retrieve a room', async () => {
    const room = {
      room_id: 'room-1',
      name: 'General',
      last_read_message_id: 0,
      created_at: new Date().toISOString(),
    };

    await db.rooms.add(room);
    const storedRoom = await db.rooms.get('room-1');

    expect(storedRoom).toEqual(room);
  });

  it('should store and retrieve messages', async () => {
    const message = {
      room_id: 'room-1',
      message_id: 1,
      sender_id: 'user-1',
      content: 'Hello World',
      created_at: new Date().toISOString(),
    };

    await db.messages.add(message);
    const storedMessage = await db.messages.get({ room_id: 'room-1', message_id: 1 });

    expect(storedMessage).toEqual(message);
  });
});
