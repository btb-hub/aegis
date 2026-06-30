import { render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { LoginPage } from './LoginPage';

function renderLoginPage() {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <AuthProvider>
          <LoginPage />
        </AuthProvider>
      </MemoryRouter>
    </I18nextProvider>,
  );
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders provider sign-in links when unsigned', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: async () => ({}),
    } as Response);

    renderLoginPage();

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'Sign in with Google' })).toHaveAttribute(
        'href',
        '/auth/google/login',
      );
    });
    expect(screen.getByRole('link', { name: 'Sign in with Slack' })).toHaveAttribute(
      'href',
      '/auth/slack/login',
    );
    expect(screen.getByRole('link', { name: 'Sign in with eXpress' })).toHaveAttribute(
      'href',
      '/auth/express/login',
    );
  });
});
