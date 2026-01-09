import { useState, useEffect } from 'react';
import { useLiveQuery } from 'dexie-react-hooks';
import { db } from './db';
import { syncData, createRoom } from './sync';
import { Login } from './Login';
import { AuthProvider, useAuth } from './AuthContext';
import { CreateRoomModal } from './CreateRoomModal';
import { ProfileSettingsModal } from './ProfileSettingsModal';
import { CreateUserModal } from './CreateUserModal';
import { ChatRoom } from './ChatRoom';
import {
  PlusIcon,
  ChevronRightIcon,
  SettingsIcon,
  UserPlusIcon
} from './Icons';
import './App.css';

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
