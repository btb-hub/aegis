import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { AccountPage } from './AccountPage';

const baseUser = {
  id: 'user-1',
  email: 'alice@example.com',
  display_name: 'Alice',
  role: 'member',
  locale: 'en',
  provider: 'google',
  identities: [{ provider: 'google', linked_at: '2026-01-01T00:00:00Z' }],
};

function renderPage() {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <AuthProvider>
          <AccountPage />
        </AuthProvider>
      </MemoryRouter>
    </I18nextProvider>,
  );
}

describe('AccountPage', () => {
  beforeEach(async () => {
    vi.stubGlobal('fetch', vi.fn());
    await i18n.changeLanguage('en');
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('saves display name', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/auth/me') && init?.method === 'PATCH') {
        return {
          ok: true,
          json: async () => ({ ...baseUser, display_name: 'Alice Updated' }),
        } as Response;
      }
      if (url.includes('/auth/me')) {
        return { ok: true, json: async () => baseUser } as Response;
      }
      return { ok: false, status: 401, json: async () => ({}) } as Response;
    });

    renderPage();
    await waitFor(() => {
      expect(screen.getByDisplayValue('Alice')).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText('Display name'), { target: { value: 'Alice Updated' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(screen.getByText('Profile updated')).toBeInTheDocument();
    });
  });

  it('updates locale from account page', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/auth/me') && init?.method === 'PATCH') {
        return {
          ok: true,
          json: async () => ({ ...baseUser, locale: 'ru' }),
        } as Response;
      }
      if (url.includes('/auth/me')) {
        return { ok: true, json: async () => baseUser } as Response;
      }
      return { ok: false, status: 401, json: async () => ({}) } as Response;
    });

    renderPage();
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Account' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Русский' }));

    await waitFor(() => {
      expect(screen.getByText('Язык обновлён')).toBeInTheDocument();
    });
  });

  it('generates express link code', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return { ok: true, json: async () => baseUser } as Response;
      }
      if (url.includes('/express-link-code') && init?.method === 'POST') {
        return { ok: true, json: async () => ({ code: 'abc', command: '/link abc' }) } as Response;
      }
      return { ok: false, status: 401, json: async () => ({}) } as Response;
    });

    renderPage();
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Generate link code' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Generate link code' }));
    await waitFor(() => {
      expect(screen.getByText('/link abc')).toBeInTheDocument();
    });
  });

  it('prompts sign in when session missing', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({}),
    } as Response);

    renderPage();
    expect(await screen.findByText(/Sign in to manage your account/i)).toBeInTheDocument();
  });
});
