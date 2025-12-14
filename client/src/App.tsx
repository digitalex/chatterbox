import { useState, useEffect } from 'react';
import { useLiveQuery } from 'dexie-react-hooks';
import { format } from 'date-fns';
import { db } from './db';
import { syncData, sendMessage, USER_ID } from './sync';
import { Login } from './Login';
import { AuthProvider, useAuth } from './AuthContext';
import './App.css';

// 1. Main Component
function ChatterboxApp() {
  const { user, isLoading } = useAuth();
  const [activeRoomId, setActiveRoomId] = useState<string | null>(null);

  useEffect(() => {
    syncData();
    const interval = setInterval(syncData, 5000);
    return () => clearInterval(interval);
  }, []);

  const rooms = useLiveQuery(async () => await db.rooms.toArray());

  // --- MOBILE LOGIC ---
  // If a room is active, add the class to hide the sidebar on mobile
  const sidebarClass = activeRoomId ? 'sidebar hidden-on-mobile' : 'sidebar';

  if (isLoading) return <div className="loading-screen">Loading...</div>;
  if (!user) return <Login />;

  return (
    <div className="app-container">
      {/* Apply the dynamic class here */}
      <aside className={sidebarClass}>
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
          <ChatRoom 
            roomId={activeRoomId} 
            // Pass the handler to clear the active room (showing the sidebar again)
            onBack={() => setActiveRoomId(null)} 
          />
        ) : (
          <div className="empty-state">Select a room to start chatting</div>
        )}
      </main>
    </div>
  );
}

// 2. Export Wrapper
export default function App() {
  return (
    <AuthProvider>
      <ChatterboxApp />
    </AuthProvider>
  );
}

// 3. ChatRoom Component
interface ChatRoomProps {
  roomId: string;
  onBack: () => void;
}

function ChatRoom({ roomId, onBack }: ChatRoomProps) {
    const messages = useLiveQuery(
      () => db.messages.where('room_id').equals(roomId).sortBy('created_at'),
      [roomId]
    );

    const userMap = useLiveQuery(async () => {
      const users = await db.users.toArray();
      return new Map(users.map(u => [u.user_id, u.display_name]));
    });

    const [inputText, setInputText] = useState('');
  
    const handleSend = async () => {
      if (!inputText.trim()) return;
      const textToSend = inputText;
      setInputText('');
      await sendMessage(roomId, { text: textToSend });
    };

    const roomName = useLiveQuery(() => db.rooms.get(roomId))?.name || 'Chat';
  
    return (
      <div className="room-view">
        <div className="chat-header">
          {/* This button is hidden on desktop by CSS, shown on mobile */}
          <button className="back-button" onClick={onBack}>
            &#8592; {/* Left Arrow Character */}
          </button>
          <span className="chat-title">#{roomName}</span>
        </div>
        
        <div className="message-list">
          {messages?.map((msg) => (
            <div key={msg.message_id} className={`message-bubble ${msg.sender_id === USER_ID ? 'my-message' : ''}`}>
              <div className="meta">
                <span className="author">
                  {userMap?.get(msg.sender_id) || msg.sender_id}
                </span>
                <span className="time">{format(new Date(msg.created_at), 'HH:mm')}</span>
              </div>
              <div className="body">
                {(msg.content && typeof msg.content === 'object' && 'text' in msg.content) 
                  ? (msg.content as any).text 
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