import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthProvider, useAuth } from '../AuthContext';
import { API_URL, setAuthInfo, clearAuthInfo } from '../sync';

// Mock fetch
const fetchMock = vi.fn();
vi.stubGlobal('fetch', fetchMock);

// Mock Token (valid until year 2286)
// Header: {"alg":"HS256","typ":"JWT"} -> eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
// Payload: {"user_id":"test-user-id","exp":9999999999} -> eyJ1c2VyX2lkIjoidGVzdC11c2VyLWlkIiwiZXhwIjo5OTk5OTk5OTk5fQ
const MOCK_TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidGVzdC11c2VyLWlkIiwiZXhwIjo5OTk5OTk5OTk5fQ.signature";
const MOCK_USER_ID = "test-user-id";

// Test component to consume the context
const TestComponent = () => {
  const { user, login, logout, isLoading } = useAuth();

  if (isLoading) return <div>Loading...</div>;

  return (
    <div>
      {user ? (
        <>
          <div data-testid="user-info">{user.name} ({user.id})</div>
          <button onClick={logout}>Logout</button>
        </>
      ) : (
        <button onClick={() => login('testuser', 'password')}>Login</button>
      )}
    </div>
  );
};

describe('AuthContext', () => {
  beforeEach(() => {
    fetchMock.mockReset();
    clearAuthInfo(); // Clears sync.ts state and localStorage
    vi.resetModules();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('should initialize with no user if localStorage is empty', async () => {
    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    expect(await screen.findByText('Login')).toBeInTheDocument();
    expect(screen.queryByTestId('user-info')).not.toBeInTheDocument();
  });

  it('should initialize with user if localStorage has valid token', async () => {
    setAuthInfo(MOCK_USER_ID, MOCK_TOKEN);
    localStorage.setItem('chatterbox_username', 'Saved User');

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    expect(await screen.findByTestId('user-info')).toHaveTextContent(`Saved User (${MOCK_USER_ID})`);
    expect(screen.queryByText('Login')).not.toBeInTheDocument();
  });

  it('should login successfully', async () => {
    // 1. Mock /login call
    fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ token: MOCK_TOKEN }),
    });

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    const loginButton = await screen.findByText('Login');
    await userEvent.click(loginButton);

    // Verify /login called
    expect(fetchMock).toHaveBeenNthCalledWith(1, `${API_URL}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: 'testuser', password: 'password' }),
    });

    expect(localStorage.getItem('chatterbox_username')).toBe('testuser');
    expect(await screen.findByTestId('user-info')).toHaveTextContent(`testuser (${MOCK_USER_ID})`);
  });

  it('should logout successfully', async () => {
    setAuthInfo(MOCK_USER_ID, MOCK_TOKEN);
    localStorage.setItem('chatterbox_username', 'Saved User');

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    const logoutButton = await screen.findByText('Logout');
    await userEvent.click(logoutButton);

    expect(localStorage.getItem('chatterbox_username')).toBeNull();
    expect(localStorage.getItem('chatterbox_auth_token')).toBeNull();
    expect(await screen.findByText('Login')).toBeInTheDocument();
  });
});
