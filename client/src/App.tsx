import { useState, useEffect } from 'react';
import { useLiveQuery } from 'dexie-react-hooks';
import { format } from 'date-fns';
import { db } from './db';
import { syncData, sendMessage, createRoom, USER_ID } from './sync';
import { Login } from './Login';
import { AuthProvider, useAuth } from './AuthContext';
import { CreateRoomModal } from './CreateRoomModal';
import './App.css';

// --- ICONS (Inline SVGs) ---
const PlusIcon = () => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
);
const ChevronRightIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#ccc" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>
);
const BackIcon = () => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>
);
const InfoIcon = () => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor" stroke="none"><circle cx="12" cy="12" r="10" /><path stroke="white" strokeWidth="2" d="M12 16v-4M12 8h.01" /></svg>
);
const SmileyIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#888" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"></circle><path d="M8 14s1.5 2 4 2 4-2 4-2"></path><line x1="9" y1="9" x2="9.01" y2="9"></line><line x1="15" y1="9" x2="15.01" y2="9"></line></svg>
);
const SendIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="black" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg>
);
const CheckIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#999" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
);

// 1. Main Component
function ChatterboxApp() {
  const { user, isLoading } = useAuth();
  const [activeRoomId, setActiveRoomId] = useState<string | null>(null);
  const [isNewRoomModalOpen, setIsNewRoomModalOpen] = useState(false);

  useEffect(() => {
    syncData();
    const interval = setInterval(syncData, 5000);
    return () => clearInterval(interval);
  }, []);

  const rooms = useLiveQuery(async () => await db.rooms.toArray());

  const handleCreateRoom = async (name: string) => {
    // createRoom now adds to DB internally
    const room = await createRoom(name);
    setActiveRoomId(room.room_id);
  };

  const sidebarClass = activeRoomId ? 'sidebar hidden-on-mobile' : 'sidebar';

  if (isLoading) return <div className="loading-screen">Loading...</div>;
  if (!user) return <Login />;

  return (
    <div className="app-container">
      {isNewRoomModalOpen && (
        <CreateRoomModal
          existingNames={rooms?.map((r) => r.name) || []}
          onClose={() => setIsNewRoomModalOpen(false)}
          onCreate={handleCreateRoom}
        />
      )}
      <aside className={sidebarClass}>
        <div className="sidebar-header">
           <h1>Lobby</h1>
           <button onClick={() => setIsNewRoomModalOpen(true)} className="new-room-btn">
             <PlusIcon />
           </button>
        </div>

        <div className="sidebar-subheader">
          <span>YOUR ROOMS</span>
          <a href="#" className="see-all">See all</a>
        </div>

        <div className="room-list">
          {rooms?.map((room) => (
            <div
              key={room.room_id}
              className={`room-card ${activeRoomId === room.room_id ? 'active' : ''}`}
              onClick={() => setActiveRoomId(room.room_id)}
            >
              <div className="room-status-indicator"></div>
              <div className="room-info">
                <div className="room-name-row">
                  <span className="room-name">{room.name}</span>
                </div>
                <div className="room-members">3 members</div>
              </div>
              <div className="room-meta">
                <span className="room-time">10m</span>
                <ChevronRightIcon />
              </div>
            </div>
          ))}
        </div>
      </aside>

      <main className="chat-window">
        {activeRoomId ? (
          <ChatRoom 
            roomId={activeRoomId} 
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

    const roomName = useLiveQuery(
      () => db.rooms.get(roomId),
      [roomId]
    )?.name || 'Chat';
  
    return (
      <div className="room-view">
        <div className="chat-header">
          <button className="back-button" onClick={onBack}>
            <BackIcon />
          </button>
          <div className="chat-header-info">
            <span className="chat-title">{roomName}</span>
            <span className="chat-status"><span className="status-dot"></span> Active now</span>
          </div>
          <button className="info-button">
            <InfoIcon />
          </button>
        </div>
        
        <div className="message-list">
          <div className="date-separator"><span>Today</span></div>
          {messages?.map((msg) => {
            const isMe = msg.sender_id === USER_ID;
            return (
              <div key={msg.message_id} className={`message-row ${isMe ? 'outgoing' : 'incoming'}`}>
                {!isMe && <div className="avatar">{userMap?.get(msg.sender_id)?.charAt(0) || '?'}</div>}

                <div className="message-content">
                  {!isMe && <span className="author-name">{userMap?.get(msg.sender_id) || msg.sender_id}</span>}

                  <div className="message-bubble">
                    <div className="body">
                      {(msg.content && typeof msg.content === 'object' && 'text' in msg.content)
                        ? (msg.content as any).text
                        : JSON.stringify(msg.content)
                      }
                    </div>
                  </div>

                  <div className="message-time">
                    {format(new Date(msg.created_at), 'hh:mm a')}
                    {isMe && <span className="read-receipt"><CheckIcon /></span>}
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        <div className="composer-container">
          <button className="composer-btn plus-btn">
             <PlusIcon />
          </button>
          <div className="input-wrapper">
             <input
              type="text"
              placeholder="Type a message..."
              value={inputText}
              onChange={(e) => setInputText(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSend();
              }}
            />
            <button className="smiley-btn"><SmileyIcon /></button>
          </div>
          <button className="send-btn" onClick={handleSend} disabled={!inputText.trim()}>
            <SendIcon />
          </button>
        </div>
      </div>
    );
}
