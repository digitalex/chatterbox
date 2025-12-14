import { useState } from 'react';
import { useAuth } from './AuthContext'; // Import the hook
import './Login.css'; // We will add some styles below

export function Login() {
  const { login } = useAuth();
  const [name, setName] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    setIsSubmitting(true);
    await login(name);
    // No need to setSubmitting(false) or redirect, 
    // the AuthContext state change will trigger App to unmount this component.
  };

  return (
    <div className="login-wrapper">
      <div className="login-card">
        <h1>👋 Welcome</h1>
        <p>Choose a display name to join the conversation.</p>
        
        <form onSubmit={handleSubmit}>
          <input
            type="text"
            placeholder="Your Name (e.g. Alex)"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={isSubmitting}
            autoFocus
          />
          <button type="submit" disabled={isSubmitting || !name.trim()}>
            {isSubmitting ? 'Joining...' : 'Continue'}
          </button>
        </form>
        
        <div className="login-footer">
           You will remain anonymous. No password required.
        </div>
      </div>
    </div>
  );
}