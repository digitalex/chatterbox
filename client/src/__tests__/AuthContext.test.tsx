import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import { AuthProvider, useAuth } from '../AuthContext';
import { USER_ID, API_URL } from '../sync';

// Mock fetch
const fetchMock = vi.fn();
vi.stubGlobal('fetch', fetchMock);

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
        <button onClick={() => login('Test User')}>Login</button>
      )}
    </div>
  );
};

describe('AuthContext', () => {
  beforeEach(() => {
    fetchMock.mockReset();
    localStorage.clear();
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

  it('should initialize with user if localStorage has username', async () => {
    localStorage.setItem('chatterbox_username', 'Saved User');

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    expect(await screen.findByTestId('user-info')).toHaveTextContent(`Saved User (${USER_ID})`);
    expect(screen.queryByText('Login')).not.toBeInTheDocument();
  });

  it('should login successfully', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    });

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    const loginButton = await screen.findByText('Login');
    await userEvent.click(loginButton);

    expect(fetchMock).toHaveBeenCalledWith(`${API_URL}/me`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-User-ID': USER_ID },
      body: JSON.stringify({ display_name: 'Test User' }),
    });

    expect(localStorage.getItem('chatterbox_username')).toBe('Test User');
    expect(await screen.findByTestId('user-info')).toHaveTextContent(`Test User (${USER_ID})`);
  });

  it('should logout successfully', async () => {
    localStorage.setItem('chatterbox_username', 'Saved User');

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    const logoutButton = await screen.findByText('Logout');
    await userEvent.click(logoutButton);

    expect(localStorage.getItem('chatterbox_username')).toBeNull();
    expect(await screen.findByText('Login')).toBeInTheDocument();
  });
});
