import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../../context/AuthContext';
import i18n from '../../i18n';
import { LanguageSwitcher } from './LanguageSwitcher';

function renderSwitcher() {
  return render(
    <I18nextProvider i18n={i18n}>
      <AuthProvider>
        <LanguageSwitcher />
      </AuthProvider>
    </I18nextProvider>,
  );
}

describe('LanguageSwitcher', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({}),
    }));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders both languages', () => {
    renderSwitcher();
    expect(screen.getByRole('button', { name: 'English' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Русский' })).toBeInTheDocument();
  });

  it('switches locale to Russian', async () => {
    await i18n.changeLanguage('en');
    renderSwitcher();
    fireEvent.click(screen.getByRole('button', { name: 'Русский' }));
    expect(localStorage.getItem('aegis_locale')).toBe('ru');
    expect(i18n.language.startsWith('ru')).toBe(true);
    await i18n.changeLanguage('en');
  });

  it('syncs locale to API when user is signed in', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).includes('/auth/me') && init?.method === 'PATCH') {
        return { ok: true, json: async () => ({ locale: 'ru' }) } as Response;
      }
      if (String(input).includes('/auth/me')) {
        return {
          ok: true,
          json: async () => ({
            id: 'user-1',
            email: 'a@example.com',
            display_name: 'Alice',
            role: 'member',
            locale: 'en',
            provider: 'google',
          }),
        } as Response;
      }
      return { ok: false, status: 401, json: async () => ({}) } as Response;
    });

    renderSwitcher();
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith('/auth/me', expect.anything());
    });

    fireEvent.click(screen.getByRole('button', { name: 'Русский' }));
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        '/auth/me',
        expect.objectContaining({ method: 'PATCH' }),
      );
    });
  });
});
