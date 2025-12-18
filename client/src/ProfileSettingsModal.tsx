import { useState, useEffect } from 'react';
import { useAuth } from './AuthContext';
import { db } from './db';
import { API_URL, getAuthToken } from './sync';

interface ProfileSettingsModalProps {
  onClose: () => void;
}

export function ProfileSettingsModal({ onClose }: ProfileSettingsModalProps) {
  const { user } = useAuth();
  const [displayName, setDisplayName] = useState('');
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  const [activeTab, setActiveTab] = useState<'profile' | 'security'>('profile');
  const [statusMsg, setStatusMsg] = useState('');
  const [errorMsg, setErrorMsg] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (user?.id) {
      db.users.get(user.id).then(u => {
        if (u && u.display_name) {
          setDisplayName(u.display_name);
        } else {
             setDisplayName(user.name);
        }
      });
    }
  }, [user]);

  const handleUpdateProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setStatusMsg('');
    setErrorMsg('');
    setIsSubmitting(true);

    try {
      const token = getAuthToken();
      const res = await fetch(`${API_URL}/me`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ display_name: displayName })
      });

      if (!res.ok) throw new Error('Failed to update profile');

      if (user?.id) {
          await db.users.update(user.id, { display_name: displayName });
      }

      setStatusMsg('Profile updated successfully');
    } catch (err) {
      setErrorMsg('Error updating profile');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setStatusMsg('');
    setErrorMsg('');

    if (newPassword !== confirmPassword) {
        setErrorMsg("New passwords don't match");
        return;
    }

    setIsSubmitting(true);

    try {
      const token = getAuthToken();
      const res = await fetch(`${API_URL}/change-password`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ old_password: oldPassword, new_password: newPassword })
      });

      if (!res.ok) {
          const text = await res.text();
          // Try to parse JSON error if possible
          try {
             const jsonErr = JSON.parse(text);
             if (jsonErr.message) throw new Error(jsonErr.message);
          } catch (e) {
             // ignore
          }
          throw new Error(text || 'Failed to change password');
      }

      setStatusMsg('Password changed successfully');
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err: any) {
      setErrorMsg(err.message || 'Error changing password');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content settings-modal">
        <div className="modal-header">
            <h2>Settings</h2>
            <button className="close-btn" onClick={onClose}>&times;</button>
        </div>

        <div className="tabs">
            <button
                className={activeTab === 'profile' ? 'active' : ''}
                onClick={() => { setActiveTab('profile'); setStatusMsg(''); setErrorMsg(''); }}
            >
                Profile
            </button>
            <button
                className={activeTab === 'security' ? 'active' : ''}
                onClick={() => { setActiveTab('security'); setStatusMsg(''); setErrorMsg(''); }}
            >
                Security
            </button>
        </div>

        <div className="modal-body">
            {statusMsg && <div className="success-msg">{statusMsg}</div>}
            {errorMsg && <div className="error-msg">{errorMsg}</div>}

            {activeTab === 'profile' && (
                <form onSubmit={handleUpdateProfile}>
                    <div className="form-group">
                        <label>Display Name</label>
                        <input
                            type="text"
                            value={displayName}
                            onChange={e => setDisplayName(e.target.value)}
                            disabled={isSubmitting}
                        />
                    </div>
                    <button type="submit" className="btn-primary" disabled={isSubmitting}>Save Changes</button>
                </form>
            )}

            {activeTab === 'security' && (
                <form onSubmit={handleChangePassword}>
                    <div className="form-group">
                        <label>Current Password</label>
                        <input
                            type="password"
                            value={oldPassword}
                            onChange={e => setOldPassword(e.target.value)}
                            disabled={isSubmitting}
                        />
                    </div>
                    <div className="form-group">
                        <label>New Password</label>
                        <input
                            type="password"
                            value={newPassword}
                            onChange={e => setNewPassword(e.target.value)}
                            disabled={isSubmitting}
                        />
                    </div>
                    <div className="form-group">
                        <label>Confirm New Password</label>
                        <input
                            type="password"
                            value={confirmPassword}
                            onChange={e => setConfirmPassword(e.target.value)}
                            disabled={isSubmitting}
                        />
                    </div>
                    <button type="submit" className="btn-primary" disabled={isSubmitting}>Update Password</button>
                </form>
            )}
        </div>
      </div>
    </div>
  );
}
