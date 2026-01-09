import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { ChatRoom } from '../ChatRoom';
import { db } from '../db';
import { AuthProvider } from '../AuthContext';
import { setAuthInfo, clearAuthInfo } from '../sync';

// Mock scrollTo since it's not implemented in JSDOM
Element.prototype.scrollTo = vi.fn();

// Mock dependencies
vi.mock('../sync', async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...actual,
    sendMessage: vi.fn(),
    syncData: vi.fn(),
  };
});

const MOCK_TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidGVzdC11c2VyLWlkIiwiZXhwIjo5OTk5OTk5OTk5fQ.signature";
const MOCK_USER_ID = "test-user-id";
const ROOM_ID = "room-1";

describe('ChatRoom Pagination', () => {
  beforeEach(async () => {
    // Setup DB
    await db.delete();
    await db.open();

    // Setup Auth
    clearAuthInfo();
    setAuthInfo(MOCK_USER_ID, MOCK_TOKEN);
    localStorage.setItem('chatterbox_username', 'Test User');

    // Create Room
    await db.rooms.add({
      room_id: ROOM_ID,
      name: 'Test Room',
      last_read_message_id: 0,
      created_at: new Date().toISOString(),
      synced: 1
    });

    // Seed 150 messages
    const messages = [];
    for (let i = 1; i <= 150; i++) {
      messages.push({
        room_id: ROOM_ID,
        message_id: i,
        sender_id: MOCK_USER_ID,
        content: { text: `Message ${i}` },
        created_at: new Date(Date.now() + i * 1000).toISOString(),
        synced: 1
      });
    }
    await db.messages.bulkAdd(messages);
  });

  afterEach(async () => {
    vi.clearAllMocks();
  });

  it('should initially load only 100 messages', async () => {
    render(
      <AuthProvider>
        <ChatRoom roomId={ROOM_ID} onBack={() => {}} />
      </AuthProvider>
    );

    // Wait for messages to load
    // Message 150 is the newest, Message 1 is the oldest.
    // Default limit 100 should show messages 51 to 150.

    // Check if newest message is present
    expect(await screen.findByText('Message 150')).toBeInTheDocument();

    // Check if 51st message is present
    expect(await screen.findByText('Message 51')).toBeInTheDocument();

    // Check if oldest message (1) is NOT present
    expect(screen.queryByText('Message 1')).not.toBeInTheDocument();

    // Verify count
    const rows = document.querySelectorAll('.message-row');
    expect(rows.length).toBe(100);
  });

  it('should load more messages when scrolling up', async () => {
    render(
      <AuthProvider>
        <ChatRoom roomId={ROOM_ID} onBack={() => {}} />
      </AuthProvider>
    );

    // Wait for initial load
    expect(await screen.findByText('Message 150')).toBeInTheDocument();

    const messageList = document.querySelector('.message-list');
    expect(messageList).not.toBeNull();

    // Simulate scroll to top
    // Trigger scroll event
    fireEvent.scroll(messageList!, { target: { scrollTop: 0 } });

    // Wait for update
    // Should load more. Assuming +100 or +50.
    // If +50, we should see Message 1.
    // If limit becomes 150, we see all.

    // We need to wait for the UI to update. findByText is good.
    expect(await screen.findByText('Message 1')).toBeInTheDocument();

    const rows = document.querySelectorAll('.message-row');
    expect(rows.length).toBe(150);
  });
});
