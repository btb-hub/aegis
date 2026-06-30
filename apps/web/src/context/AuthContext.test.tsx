import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../context/AuthContext';
import i18n from '../i18n';

function AuthProbe() {
  const { user, loading, signOut } = useAuth();
  if (loading) {
    return <div>loading</div>;
  }
  return (
    <div>
      <span>{user ? user.display_name : 'signed-out'}</span>
      <button type="button" onClick={() => void signOut()}>
        sign-out-action
      </button>
    </div>
  );
}

function renderAuthProvider() {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <AuthProvider>
          <AuthProbe />
        </AuthProvider>
      </MemoryRouter>
    </I18nextProvider>,
  );
}

describe('AuthProvider', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('loads signed-in user from /auth/me', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        id: 'user-1',
        email: 'alice@example.com',
        display_name: 'Alice',
        role: 'admin',
        locale: 'en',
        provider: 'google',
      }),
    } as Response);

    renderAuthProvider();

    await waitFor(() => {
      expect(screen.getByText('Alice')).toBeInTheDocument();
    });
  });

  it('treats 401 as signed out', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: async () => ({}),
    } as Response);

    renderAuthProvider();

    await waitFor(() => {
      expect(screen.getByText('signed-out')).toBeInTheDocument();
    });
  });

  it('calls POST /auth/logout on sign out', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          id: 'user-1',
          email: 'alice@example.com',
          display_name: 'Alice',
          role: 'admin',
          locale: 'en',
          provider: 'google',
        }),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        status: 204,
        json: async () => ({}),
      } as Response);

    renderAuthProvider();

    await waitFor(() => {
      expect(screen.getByText('Alice')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'sign-out-action' }));

    await waitFor(() => {
      expect(screen.getByText('signed-out')).toBeInTheDocument();
    });
    expect(fetch).toHaveBeenLastCalledWith('/auth/logout', {
      method: 'POST',
      credentials: 'include',
    });
  });
});
