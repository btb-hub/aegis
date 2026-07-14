import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../../context/AuthContext';
import i18n from '../../i18n';
import { AppShell } from './AppShell';

function renderShell(ui: ReactNode) {
  vi.mocked(fetch).mockResolvedValue({
    ok: false,
    status: 401,
    json: async () => ({}),
  } as Response);

  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <AuthProvider>{ui}</AuthProvider>
      </MemoryRouter>
    </I18nextProvider>,
  );
}

describe('AppShell', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders shell chrome', () => {
    renderShell(
      <AppShell>
        <div>content</div>
      </AppShell>,
    );
    expect(screen.getByText('content')).toBeInTheDocument();
    expect(screen.getAllByText('Shifts').length).toBeGreaterThan(0);
  });

  it('renders Russian navigation when locale is ru', async () => {
    await i18n.changeLanguage('ru');
    renderShell(
      <AppShell>
        <div>x</div>
      </AppShell>,
    );
    expect(screen.getByText('Смены')).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });

  it('calls onNavigate when a nav item is clicked', () => {
    const onNavigate = vi.fn();
    renderShell(
      <AppShell currentPage="shifts" onNavigate={onNavigate}>
        <div>content</div>
      </AppShell>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Incidents' }));
    expect(onNavigate).toHaveBeenCalledWith('incidents');
  });

  it('includes Workspaces in navigation', () => {
    renderShell(
      <AppShell currentPage="workspaces" onNavigate={vi.fn()}>
        <div>content</div>
      </AppShell>,
    );
    expect(screen.getByRole('button', { name: 'Workspaces' })).toBeInTheDocument();
  });

  it('shows signed-in user and sign out button', () => {
    const onSignOut = vi.fn().mockResolvedValue(undefined);
    renderShell(
      <AppShell
        user={{
          id: 'user-1',
          email: 'alice@example.com',
          display_name: 'Alice',
          role: 'admin',
          locale: 'en',
          provider: 'google',
        }}
        onSignOut={onSignOut}
      >
        <div>content</div>
      </AppShell>,
    );

    expect(screen.getByRole('link', { name: 'Alice' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeInTheDocument();
  });

  it('calls sign out and navigates to login', async () => {
    const onSignOut = vi.fn().mockResolvedValue(undefined);
    renderShell(
      <AppShell
        user={{
          id: 'user-1',
          email: 'alice@example.com',
          display_name: 'Alice',
          role: 'admin',
          locale: 'en',
          provider: 'google',
        }}
        onSignOut={onSignOut}
      >
        <div>content</div>
      </AppShell>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Sign out' }));

    await waitFor(() => {
      expect(onSignOut).toHaveBeenCalled();
    });
  });
});
