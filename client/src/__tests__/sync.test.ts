import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { syncData, sendMessage, createRoom, setAuthInfo, clearAuthInfo, API_URL } from '../sync';
import { db } from '../db';

// Mock fetch
const fetchMock = vi.fn();
vi.stubGlobal('fetch', fetchMock);

const MOCK_TOKEN = "mock-token";
const MOCK_USER_ID = "test-user";

describe('Sync Logic', () => {
  beforeEach(async () => {
    fetchMock.mockReset();
    await db.delete();
    await db.open();
    await db.config.clear();

    // Set Auth Info
    setAuthInfo(MOCK_USER_ID, MOCK_TOKEN);
  });

  afterEach(() => {
    vi.clearAllMocks();
    clearAuthInfo();
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

    // Should call /sync directly with token
    expect(fetchMock).toHaveBeenCalledWith(`${API_URL}/sync`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${MOCK_TOKEN}`,
      },
      body: expect.stringContaining('"last_synced_at":null'),
    });

    const rooms = await db.rooms.toArray();
    expect(rooms).toHaveLength(1);
  });

  it('should include last_synced_at in request if available', async () => {
    await db.config.put({ key: 'last_synced_at', value: '2023-01-01T00:00:00Z' });

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    });

    await syncData();

    expect(fetchMock).toHaveBeenCalledWith(`${API_URL}/sync`, expect.objectContaining({
      body: expect.stringContaining('"last_synced_at":"2023-01-01T00:00:00Z"'),
    }));
  });

  it('should send a message and trigger sync', async () => {
    // Mock Sync Response
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    });

    await sendMessage('room-1', 'Hello');

    // Wait for async syncData to potentially run
    await new Promise(r => setTimeout(r, 10));

    const messages = await db.messages.toArray();
    expect(messages).toHaveLength(1);
    // Sync should have completed successfully
    expect(messages[0].synced).toBe(1);

    // Should trigger sync
    expect(fetchMock).toHaveBeenCalledWith(`${API_URL}/sync`, expect.anything());
  });

  it('should create a room and trigger sync', async () => {
    // Mock Sync Response
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    });

    await createRoom('New Room');

    // Wait for async syncData to potentially run
    await new Promise(r => setTimeout(r, 10));

    const rooms = await db.rooms.toArray();
    expect(rooms).toHaveLength(1);
    expect(rooms[0].name).toBe('New Room');

    // Should trigger sync
    expect(fetchMock).toHaveBeenCalledWith(`${API_URL}/sync`, expect.anything());
  });

  it('should skip sync silently if not logged in', async () => {
    clearAuthInfo();
    const consoleSpy = vi.spyOn(console, 'error');

    await syncData();

    expect(fetchMock).not.toHaveBeenCalled();
    expect(consoleSpy).not.toHaveBeenCalled();
  });
});
