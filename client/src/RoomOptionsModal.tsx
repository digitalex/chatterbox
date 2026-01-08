import { useState } from 'react';
import { useAuth } from './AuthContext';
import { API_URL, getAuthToken } from './sync';

interface RoomOptionsModalProps {
  room: { room_id: string; name: string };
  onClose: () => void;
  onUpdate: () => void; // Trigger refresh
  onDelete: () => void; // Trigger navigation back
}

export function RoomOptionsModal({ room, onClose, onUpdate, onDelete }: RoomOptionsModalProps) {
  const { user } = useAuth();
  const [newName, setNewName] = useState(room.name);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!user?.is_admin) return null;

  const handleRename = async () => {
    try {
      const token = await getAuthToken();
      const res = await fetch(`${API_URL}/rooms/${room.room_id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ name: newName })
      });

      if (!res.ok) throw new Error('Failed to rename room');

      onUpdate();
      onClose();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleDelete = async () => {
    try {
      const token = await getAuthToken();
      const res = await fetch(`${API_URL}/rooms/${room.room_id}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (!res.ok) throw new Error('Failed to delete room');

      onDelete();
      onClose();
    } catch (err: any) {
      setError(err.message);
    }
  };

  if (isDeleteConfirmOpen) {
    return (
      <div className="modal-overlay">
        <div className="modal-content">
          <h2>Delete Room?</h2>
          <p>Are you sure you want to delete <strong>{room.name}</strong>? This cannot be undone.</p>
          {error && <p className="error-message">{error}</p>}
          <div className="modal-actions">
            <button onClick={() => setIsDeleteConfirmOpen(false)} className="btn-cancel">Cancel</button>
            <button onClick={handleDelete} className="btn-danger">Delete</button>
          </div>
        </div>
      </div>
    );
  }

  const isNameChanged = newName.trim() !== room.name && newName.trim() !== "";

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h2>Room Options</h2>
        {error && <p className="error-message">{error}</p>}

        <div className="modal-body" style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>

          <div className="rename-row" style={{ display: 'flex', gap: '10px' }}>
             <input
               type="text"
               value={newName}
               onChange={(e) => setNewName(e.target.value)}
               className="modal-input"
               style={{ flex: 1 }}
             />
             <button
               onClick={handleRename}
               disabled={!isNameChanged}
               className="btn-primary"
               style={{ opacity: isNameChanged ? 1 : 0.5, cursor: isNameChanged ? 'pointer' : 'not-allowed', width: 'auto' }}
             >
               Rename
             </button>
          </div>

          <div className="delete-row">
            <button
              onClick={() => setIsDeleteConfirmOpen(true)}
              className="btn-danger"
              style={{ width: '100%' }}
            >
              Delete room
            </button>
          </div>

        </div>

        <div className="modal-actions" style={{ marginTop: '20px' }}>
          <button onClick={onClose} className="btn-cancel">Close</button>
        </div>
      </div>
    </div>
  );
}
