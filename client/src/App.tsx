import { useState, useEffect } from 'react';
import { useLiveQuery } from 'dexie-react-hooks';
import { db, type Room, type Message } from './db';
import { syncData, sendMessage } from './sync';
import { format } from 'date-fns';
import './App.css'; // You'll need some basic CSS
import { Login } from './Login';

function App() {
  const [hasJoined, setHasJoined] = useState(false);
  const [activeRoomId, setActiveRoomId] = useState<string | null>(null);

  // 1. Check if we are already logged in on mount
  useEffect(() => {
    const savedName = localStorage.getItem('chatterbox_username');
    if (savedName) setHasJoined(true);
    
    // Start sync immediately regardless, so data is ready
    syncData();
    const interval = setInterval(syncData, 5000);
    return () => clearInterval(interval);
  }, []);

  // 2. Auto-Sync on mount and every 5 seconds
  useEffect(() => {
    syncData(); // Initial load
    const interval = setInterval(syncData, 5000);
    return () => clearInterval(interval);
  }, []);

  // 2. Live Query: Get Rooms
  const rooms = useLiveQuery(async () => {
    return await db.rooms.toArray();
  });

  // Show Login Screen if not joined
  if (!hasJoined) {
    return <Login onLogin={() => setHasJoined(true)} />;
  }


  return (
    <div className="app-container">
      {/* Sidebar */}
      <aside className="sidebar">
        <div className="room-list">
          {rooms?.map((room) => (
            <div
              key={room.room_id}
              className={`room-item ${activeRoomId === room.room_id ? 'active' : ''}`}
              onClick={() => setActiveRoomId(room.room_id)}
            >
              #{room.name}
              {/* Simple unread dot logic could go here later */}
            </div>
          ))}
        </div>
      </aside>

      {/* Main Chat Area */}
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

// Sub-component for the active chat
function ChatRoom({ roomId }: { roomId: string }) {
  // Live Query: Get messages for THIS room, sorted by time
  const messages = useLiveQuery(
    () => db.messages.where('room_id').equals(roomId).sortBy('created_at'),
    [roomId]
  );

  // State for the input
  const [inputText, setInputText] = useState('');

  const handleSend = async () => {
    if (!inputText.trim()) return;
    
    // Send plain text for now (we'll add encryption later)
    await sendMessage(roomId, { text: inputText });
    setInputText('');
  };

  return (
    <div className="room-view">
      <div className="message-list">
        {messages?.map((msg) => (
          <div key={msg.message_id} className="message-bubble">
            <div className="meta">
              <span className="author">{msg.sender_id}</span>
              <span className="time">{format(new Date(msg.created_at), 'HH:mm')}</span>
            </div>
            {/* Handle JSON content structure */}
            <div className="body">
              {msg.content?.text || JSON.stringify(msg.content)}
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
          onKeyDown={(e) => e.key === 'Enter' && handleSend()}
        />
        <button onClick={handleSend}>Send</button>
      </div>
    </div>
  );
}

export default App;

