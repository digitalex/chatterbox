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

    // Because previous tests might have set cached authToken, we may or may not see a login call first.
    // However, in this isolated test execution (if it is the first one or if env is fresh), it will call login.
    // Vitest runs in parallel threads by default but within a file it is sequential.
    // authToken is a module-level variable in sync.ts. It persists across tests in the same file.

    // To make tests deterministic, we need to handle both cases or assume persistence.
    // Since 'should sync data from server' is the FIRST test, it WILL call login.

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ token: 'mock-token' }),
    });

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponseData,
    });

    await syncData();

    expect(fetchMock).toHaveBeenNthCalledWith(1, `${API_URL}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: USER_ID }),
    });

    expect(fetchMock).toHaveBeenNthCalledWith(2, `${API_URL}/sync`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer mock-token',
      },
      body: JSON.stringify({ last_synced_at: null }),
    });

    const rooms = await db.rooms.toArray();
    expect(rooms).toHaveLength(1);
  });

  it('should include last_synced_at in request if available', async () => {
    // authToken should be cached from previous test ('mock-token').
    await db.config.put({ key: 'last_synced_at', value: '2023-01-01T00:00:00Z' });

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    });

    await syncData();

    // Should NOT call login again.
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(`${API_URL}/sync`, expect.objectContaining({
      body: JSON.stringify({ last_synced_at: '2023-01-01T00:00:00Z' }),
    }));
  });

  it('should send a message and trigger sync', async () => {
    // authToken is cached ('mock-token').

    // 1. Send Message Response
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}), // Message sent response
    });

    // 2. Sync Response (triggered by sendMessage)
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}), // Sync response
    });

    await sendMessage('room-1', 'Hello');

    expect(fetchMock).toHaveBeenNthCalledWith(1, `${API_URL}/rooms/room-1/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer mock-token',
        },
        body: JSON.stringify({ content: 'Hello' }),
    });

    expect(fetchMock).toHaveBeenNthCalledWith(2, `${API_URL}/sync`, expect.anything());
  });

  it('should create a room', async () => {
    // authToken is cached ('mock-token').
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
        'Authorization': 'Bearer mock-token',
      },
      body: JSON.stringify({ name: 'New Room' }),
    });

    expect(result).toEqual(newRoom);
  });
});
