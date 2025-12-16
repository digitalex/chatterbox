import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { syncData, sendMessage, createRoom, USER_ID, API_URL } from '../sync';
import { db } from '../db';

// Mock fetch
const fetchMock = vi.fn();
vi.stubGlobal('fetch', fetchMock);

describe('Sync Logic', () => {
  beforeEach(async () => {
    fetchMock.mockReset();
    await db.delete();
    await db.open();
    // Clear config
    await db.config.clear();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('should sync data from server', async () => {
    const mockResponseData = {
      rooms: [
        { room_id: 'room-1', name: 'General', last_read_message_id: 0, created_at: '2023-01-01T00:00:00Z' }
      ],
      messages: [
        { room_id: 'room-1', message_id: 1, sender_id: 'other-user', content: 'Hi', created_at: '2023-01-01T00:01:00Z' }
      ],
      users: [
        { user_id: 'other-user', display_name: 'Other User' }
      ],
      sync_timestamp: '2023-01-01T00:02:00Z'
    };

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponseData,
    });

    await syncData();

    // Verify fetch call
    expect(fetchMock).toHaveBeenCalledWith(`${API_URL}/sync`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': USER_ID,
      },
      body: JSON.stringify({ last_synced_at: null }),
    });

    // Verify data in DB
    const rooms = await db.rooms.toArray();
    expect(rooms).toHaveLength(1);
    expect(rooms[0].room_id).toBe('room-1');

    const messages = await db.messages.toArray();
    expect(messages).toHaveLength(1);
    expect(messages[0].content).toBe('Hi');

    const users = await db.users.toArray();
    expect(users).toHaveLength(1);
    expect(users[0].display_name).toBe('Other User');

    const lastSyncedAt = await db.config.get('last_synced_at');
    expect(lastSyncedAt?.value).toBe('2023-01-01T00:02:00Z');
  });

  it('should include last_synced_at in request if available', async () => {
    await db.config.put({ key: 'last_synced_at', value: '2023-01-01T00:00:00Z' });

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    });

    await syncData();

    expect(fetchMock).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({
      body: JSON.stringify({ last_synced_at: '2023-01-01T00:00:00Z' }),
    }));
  });

  it('should send a message and trigger sync', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}), // Message sent response
    });

    // Mock sync response
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}), // Sync response
    });

    await sendMessage('room-1', 'Hello');

    expect(fetchMock).toHaveBeenNthCalledWith(1, `${API_URL}/rooms/room-1/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': USER_ID,
        },
        body: JSON.stringify({ content: 'Hello' }),
    });

    // Check if syncData was called (implied by second fetch call)
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock).toHaveBeenNthCalledWith(2, `${API_URL}/sync`, expect.anything());
  });

  it('should create a room', async () => {
    const newRoom = { room_id: 'new-room', name: 'New Room' };
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => newRoom,
    });

    const result = await createRoom('New Room');

    expect(fetchMock).toHaveBeenCalledWith(`${API_URL}/rooms`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': USER_ID,
      },
      body: JSON.stringify({ name: 'New Room' }),
    });

    expect(result).toEqual(newRoom);
  });
});
