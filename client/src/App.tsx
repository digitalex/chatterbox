import { useState, useEffect, useRef } from 'react';
import { useLiveQuery } from 'dexie-react-hooks';
import { format } from 'date-fns';
import { db } from './db';
import { syncData, sendMessage, createRoom, USER_ID } from './sync';
import { Login } from './Login';
import { AuthProvider, useAuth } from './AuthContext';
import { CreateRoomModal } from './CreateRoomModal';
import { ProfileSettingsModal } from './ProfileSettingsModal';
import { CreateUserModal } from './CreateUserModal';
import { RoomOptionsModal } from './RoomOptionsModal';
import './App.css';

// --- ICONS (Inline SVGs) ---
const EditIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
);
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
const SettingsIcon = () => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
);
const UserPlusIcon = () => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="8.5" cy="7" r="4"></circle><line x1="20" y1="8" x2="20" y2="14"></line><line x1="23" y1="11" x2="17" y2="11"></line></svg>
);

// 1. Main Component
function ChatterboxApp() {
  const { user, isLoading } = useAuth();
  const [activeRoomId, setActiveRoomId] = useState<string | null>(null);
  const [isNewRoomModalOpen, setIsNewRoomModalOpen] = useState(false);
  const [isProfileModalOpen, setIsProfileModalOpen] = useState(false);
  const [isCreateUserModalOpen, setIsCreateUserModalOpen] = useState(false);

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
      {isProfileModalOpen && (
        <ProfileSettingsModal onClose={() => setIsProfileModalOpen(false)} />
      )}
      {isCreateUserModalOpen && (
        <CreateUserModal onClose={() => setIsCreateUserModalOpen(false)} />
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

        <div className="sidebar-footer" style={{ marginTop: 'auto', padding: '16px', borderTop: '1px solid #EEE', display: 'flex', gap: '16px', alignItems: 'center' }}>
            <button className="settings-btn" onClick={() => setIsProfileModalOpen(true)} style={{ background: 'none', border: 'none', color: '#888', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <SettingsIcon />
                <span>Settings</span>
            </button>
            {user?.is_admin && (
                <button className="create-user-btn" onClick={() => setIsCreateUserModalOpen(true)} style={{ background: 'none', border: 'none', color: '#888', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '8px' }} title="Create New User">
                    <UserPlusIcon />
                </button>
            )}
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
    const { user } = useAuth();
    const [isOptionsOpen, setIsOptionsOpen] = useState(false);
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

    const messagesEndRef = useRef<HTMLDivElement>(null);

    const scrollToBottom = () => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    };

    useEffect(() => {
      scrollToBottom();
    }, [messages]);
  
    return (
      <div className="room-view">
        {isOptionsOpen && (
          <RoomOptionsModal
            room={{room_id: roomId, name: roomName}}
            onClose={() => setIsOptionsOpen(false)}
            onUpdate={syncData}
            onDelete={() => {
               syncData();
               onBack(); // Go back to lobby
            }}
          />
        )}
        <div className="chat-header">
          <button className="back-button" onClick={onBack}>
            <BackIcon />
          </button>
          <div className="chat-header-info">
            <div style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
              <span className="chat-title">{roomName}</span>
              {user?.is_admin && (
                <button onClick={() => setIsOptionsOpen(true)} className="icon-btn" style={{padding: 0, border: 'none', background: 'none', cursor: 'pointer', color: '#666'}}>
                  <EditIcon />
                </button>
              )}
            </div>
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
          <div ref={messagesEndRef} />
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
