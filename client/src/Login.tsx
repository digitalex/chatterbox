import { useState } from 'react';
import { USER_ID, API_URL } from './sync';

export function Login({ onLogin }: { onLogin: () => void }) {
  const [username, setUsername] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    if (!username.trim()) return;
    setLoading(true);

    try {
      // 1. Tell server who we are
      await fetch(`${API_URL}/me`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': USER_ID,
        },
        body: JSON.stringify({ display_name: username }),
      });

      // 2. Save locally so we remember next time
      localStorage.setItem('chatterbox_username', username);
      
      // 3. Notify parent app
      onLogin();
    } catch (err) {
      console.error(err);
      alert('Failed to login. Check console.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      <h1>Welcome to Chatterbox</h1>
      <p>Pick a username to start chatting anonymously.</p>
      
      <input 
        type="text" 
        placeholder="e.g. Maverick" 
        value={username}
        onChange={e => setUsername(e.target.value)}
        disabled={loading}
      />
      <button onClick={handleSubmit} disabled={loading || !username}>
        {loading ? 'Joining...' : 'Join Chat'}
      </button>

      <div className="debug-id">Your ID: {USER_ID}</div>
    </div>
  );
}