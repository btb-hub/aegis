import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ProtectedRoute } from './ProtectedRoute';
import { AuthProvider } from '../../context/AuthContext';
import i18n from '../../i18n';

function renderProtectedRoute(initialPath = '/integrations') {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter initialEntries={[initialPath]}>
        <AuthProvider>
          <Routes>
            <Route path="/login" element={<div>login-page</div>} />
            <Route
              path="/integrations"
              element={
                <ProtectedRoute>
                  <div>integrations-content</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    </I18nextProvider>,
  );
}

describe('ProtectedRoute', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('redirects unsigned users to login', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: async () => ({}),
    } as Response);

    renderProtectedRoute();

    expect(await screen.findByText('login-page')).toBeInTheDocument();
  });

  it('renders children when session exists', async () => {
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

    renderProtectedRoute();

    expect(await screen.findByText('integrations-content')).toBeInTheDocument();
  });
});
