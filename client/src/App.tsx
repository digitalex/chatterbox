import { useState, useEffect } from 'react';
import { useLiveQuery } from 'dexie-react-hooks';
import { format } from 'date-fns';
import { db } from './db';
import { syncData, sendMessage, USER_ID } from './sync';
import { Login } from './Login';
import { AuthProvider, useAuth } from './AuthContext'; // Import new context
import './App.css';

// 1. We wrap the real app logic in a sub-component so it can use the hook
function ChatterboxApp() {
  const { user, isLoading } = useAuth();
  const [activeRoomId, setActiveRoomId] = useState<string | null>(null);

  // Sync Logic (Run this regardless of login state to keep DB fresh, or optionally only after login)
  useEffect(() => {
    syncData();
    const interval = setInterval(syncData, 5000);
    return () => clearInterval(interval);
  }, []);

  const rooms = useLiveQuery(async () => await db.rooms.toArray());

  // Show loading spinner while checking LocalStorage
  if (isLoading) return <div className="loading-screen">Loading...</div>;

  // Show Login if not authenticated
  if (!user) return <Login />;

  // --- Main Chat UI ---
  return (
    <div className="app-container">
      <aside className="sidebar">
        <div className="sidebar-header">
           <h2>Chatterbox</h2>
           <span className="user-badge">{user.name}</span>
        </div>
        <div className="room-list">
          {rooms?.map((room) => (
            <div
              key={room.room_id}
              className={`room-item ${activeRoomId === room.room_id ? 'active' : ''}`}
              onClick={() => setActiveRoomId(room.room_id)}
            >
              #{room.name}
            </div>
          ))}
        </div>
      </aside>

      <main className="chat-window">
        {activeRoomId ? (
          <ChatRoom roomId={activeRoomId} />
        ) : (
          <div className="empty-state">Select a room to start chatting</div>
        )}
      </main>
    </div>
  );
}

// 2. The Main Export wraps everything in the Provider
export default function App() {
  return (
    <AuthProvider>
      <ChatterboxApp />
    </AuthProvider>
  );
}

// ... (ChatRoom component stays EXACTLY the same as before) ...
function ChatRoom({ roomId }: { roomId: string }) {
    // ... Copy your existing ChatRoom code here ...
    // Note: You can replace localStorage.getItem('chatterbox_user_id') 
    // with the `USER_ID` import from './sync' for consistency.
    const messages = useLiveQuery(
      () => db.messages.where('room_id').equals(roomId).sortBy('created_at'),
      [roomId]
    );
    
    const [inputText, setInputText] = useState('');
  
    const handleSend = async () => {
      if (!inputText.trim()) return;
      const textToSend = inputText;
      setInputText('');
      await sendMessage(roomId, { text: textToSend });
    };
  
    return (
      <div className="room-view">
        <div className="message-list">
          {messages?.map((msg) => (
            <div key={msg.message_id} className={`message-bubble ${msg.sender_id === USER_ID ? 'my-message' : ''}`}>
              <div className="meta">
                <span className="author">{msg.sender_id}</span>
                <span className="time">{format(new Date(msg.created_at), 'HH:mm')}</span>
              </div>
              <div className="body">
                {(msg.content && typeof msg.content === 'object' && msg.content.text) 
                  ? msg.content.text 
                  : JSON.stringify(msg.content)
                }
              </div>
            </div>
          ))}
        </div>
        <div className="composer">
          <input 
            type="text" 
            placeholder="Type a message..." 
            value={inputText}
            onChange={(e) => setInputText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleSend();
            }}
          />
          <button onClick={handleSend} disabled={!inputText.trim()}>Send</button>
        </div>
      </div>
    );
  }