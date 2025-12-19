import React, { createContext, useContext, useState, useEffect } from 'react';
import { USER_ID, API_URL, getAuthToken, setAuthInfo, clearAuthInfo } from './sync';

// Define what our "User" looks like
interface User {
  id: string;
  name: string;
  is_admin: boolean;
}

interface AuthContextType {
  user: User | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

function parseJwt(token: string) {
    try {
        const base64Url = token.split('.')[1];
        const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
        const jsonPayload = atob(base64);
        return JSON.parse(jsonPayload);
    } catch (e) {
        return null;
    }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // 1. Check Local Storage on mount (Auto-Login)
  useEffect(() => {
    const checkAuth = async () => {
        try {
            const token = await getAuthToken(); // Throws if missing
            const userId = USER_ID;

            // Validate token expiry
            const decoded = parseJwt(token);
            if (decoded && decoded.exp * 1000 > Date.now()) {
                 const savedName = localStorage.getItem('chatterbox_username') || "User";
                 const isAdmin = decoded.is_admin || false;
                 if (userId) {
                    setUser({ id: userId, name: savedName, is_admin: isAdmin });
                 }
            } else {
                // Token expired
                logout();
            }
        } catch (e) {
            // Not logged in
        } finally {
            setIsLoading(false);
        }
    };
    checkAuth();
  }, []);

  // 2. Login Function
  const login = async (username: string, password: string) => {
    const response = await fetch(`${API_URL}/login`, {
      method: 'POST',
      headers: {
          'Content-Type': 'application/json',
      },
      body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
        throw new Error('Invalid credentials');
    }

    const data = await response.json();
    const token = data.token;

    // Decode token to get user_id
    const decoded = parseJwt(token);
    if (!decoded || !decoded.user_id) {
        throw new Error('Invalid token received');
    }

    const userId = decoded.user_id;
    const isAdmin = decoded.is_admin || false;

    // Save locally
    setAuthInfo(userId, token);
    localStorage.setItem('chatterbox_username', username);
    
    // Update State
    setUser({ id: userId, name: username, is_admin: isAdmin });
  };

  // 3. Logout Function
  const logout = () => {
    clearAuthInfo();
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
