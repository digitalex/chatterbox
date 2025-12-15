import { useState } from 'react';

interface CreateRoomModalProps {
  existingNames: string[];
  onClose: () => void;
  onCreate: (name: string) => Promise<void>;
}

export function CreateRoomModal({ existingNames, onClose, onCreate }: CreateRoomModalProps) {
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName) {
      setError('Room name cannot be empty');
      return;
    }
    if (existingNames.some(n => n.toLowerCase() === trimmedName.toLowerCase())) {
      setError('Room name already exists');
      return;
    }

    setIsSubmitting(true);
    try {
      await onCreate(trimmedName);
      onClose();
    } catch (e) {
      console.error(e);
      setError('Failed to create room');
      setIsSubmitting(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h2>Create New Room</h2>
        <form onSubmit={handleSubmit}>
          <input
            type="text"
            placeholder="Room Name"
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              setError('');
            }}
            autoFocus
            disabled={isSubmitting}
            className={error ? 'input-error' : ''}
          />
          {error && <div className="modal-error">{error}</div>}
          <div className="modal-actions">
            <button type="button" onClick={onClose} disabled={isSubmitting} className="btn-cancel">Cancel</button>
            <button type="submit" disabled={isSubmitting || !name.trim()} className="btn-create">Create</button>
          </div>
        </form>
      </div>
    </div>
  );
}
