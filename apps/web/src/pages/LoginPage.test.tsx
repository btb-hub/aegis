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

function mockAuthFetches(devEnabled: boolean) {
  vi.mocked(fetch).mockImplementation(async (input) => {
    const url = String(input);
    if (url.includes('/auth/dev/status')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ enabled: devEnabled }),
      } as Response;
    }
    return {
      ok: false,
      status: 401,
      json: async () => ({}),
    } as Response;
  });
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders provider sign-in links when unsigned', async () => {
    mockAuthFetches(false);

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
    expect(screen.queryByRole('link', { name: 'Dev sign in' })).not.toBeInTheDocument();
  });

  it('shows dev sign-in when dev auth is enabled', async () => {
    mockAuthFetches(true);

    renderLoginPage();

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'Dev sign in' })).toHaveAttribute(
        'href',
        '/auth/dev/login?role=admin',
      );
    });
    expect(screen.getByText('Local development only. Do not enable in production.')).toBeInTheDocument();
  });
});
