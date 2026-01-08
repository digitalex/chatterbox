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
  const [isRenameMode, setIsRenameMode] = useState(false);
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
            <button onClick={() => setIsDeleteConfirmOpen(false)} className="cancel-btn">Cancel</button>
            <button onClick={handleDelete} className="delete-btn" style={{backgroundColor: 'red', color: 'white'}}>Delete</button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h2>Room Options</h2>
        {error && <p className="error-message">{error}</p>}

        {!isRenameMode ? (
          <div className="modal-options">
            <button onClick={() => setIsRenameMode(true)} className="option-btn">Rename Room</button>
            <button onClick={() => setIsDeleteConfirmOpen(true)} className="option-btn delete-option" style={{color: 'red'}}>Delete Room</button>
          </div>
        ) : (
          <div className="rename-form">
            <input
              type="text"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
            />
            <div className="modal-actions">
              <button onClick={() => setIsRenameMode(false)} className="cancel-btn">Back</button>
              <button onClick={handleRename} className="save-btn">Save</button>
            </div>
          </div>
        )}

        {!isRenameMode && (
          <div className="modal-actions">
            <button onClick={onClose} className="cancel-btn">Close</button>
          </div>
        )}
      </div>
    </div>
  );
}
