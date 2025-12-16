import React, { createContext, useContext, useState, useEffect } from 'react';
import { USER_ID, API_URL } from './sync';

// Define what our "User" looks like
interface User {
  id: string;
  name: string;
}

interface AuthContextType {
  user: User | null;
  login: (name: string) => Promise<void>;
  logout: () => void;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // 1. Check Local Storage on mount (Auto-Login)
  useEffect(() => {
    const savedName = localStorage.getItem('chatterbox_username');
    if (savedName && USER_ID) {
      setUser({ id: USER_ID, name: savedName });
    }
    setIsLoading(false);
  }, []);

  // 2. Login Function
  const login = async (name: string) => {
    // Save to server (so other users see the name)
    await fetch(`${API_URL}/me`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-User-ID': USER_ID },
      body: JSON.stringify({ display_name: name }),
    });

    // Save locally
    localStorage.setItem('chatterbox_username', name);
    
    // Update State
    setUser({ id: USER_ID, name: name });
  };

  // 3. Logout Function (Optional, but good for testing)
  const logout = () => {
    localStorage.removeItem('chatterbox_username');
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, login, logout, isLoading }}>
      {children}
    </AuthContext.Provider>
  );
}

// Custom Hook for easy access
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within an AuthProvider');
  return context;
};