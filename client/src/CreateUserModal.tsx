import React, { useState } from 'react';
import { API_URL, getAuthToken } from './sync';

interface CreateUserModalProps {
  onClose: () => void;
}

export function CreateUserModal({ onClose }: CreateUserModalProps) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [isAdmin, setIsAdmin] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<boolean>(false);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const token = await getAuthToken();
      const response = await fetch(`${API_URL}/users`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          username,
          password,
          display_name: displayName || username,
          is_admin: isAdmin
        })
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || 'Failed to create user');
      }

      setSuccess(true);
      setTimeout(() => {
          onClose();
      }, 1500);

    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h2>Create New User</h2>

        {success ? (
            <div className="success-message">User created successfully!</div>
        ) : (
            <form onSubmit={handleSubmit}>
            {error && <div className="error-message">{error}</div>}

            <div className="form-group">
                <label>Username</label>
                <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                />
            </div>

            <div className="form-group">
                <label>Password</label>
                <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                />
            </div>

            <div className="form-group">
                <label>Display Name (Optional)</label>
                <input
                type="text"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                />
            </div>

            <div className="form-group checkbox-group">
                <label>
                    <input
                    type="checkbox"
                    checked={isAdmin}
                    onChange={(e) => setIsAdmin(e.target.checked)}
                    />
                    Is Admin
                </label>
            </div>

            <div className="modal-actions">
                <button type="button" onClick={onClose} disabled={loading}>Cancel</button>
                <button type="submit" disabled={loading}>
                    {loading ? 'Creating...' : 'Create User'}
                </button>
            </div>
            </form>
        )}
      </div>
    </div>
  );
}
