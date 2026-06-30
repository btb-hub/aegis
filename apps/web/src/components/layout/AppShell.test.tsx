import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { AppShell } from './AppShell';

describe('AppShell', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders shell chrome', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <AppShell>
            <div>content</div>
          </AppShell>
        </MemoryRouter>
      </I18nextProvider>,
    );
    expect(screen.getByText('content')).toBeInTheDocument();
    expect(screen.getByText('Shifts')).toBeInTheDocument();
  });

  it('renders Russian navigation when locale is ru', async () => {
    await i18n.changeLanguage('ru');
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <AppShell>
            <div>x</div>
          </AppShell>
        </MemoryRouter>
      </I18nextProvider>,
    );
    expect(screen.getByText('Смены')).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });

  it('calls onNavigate when a nav item is clicked', () => {
    const onNavigate = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <AppShell currentPage="shifts" onNavigate={onNavigate}>
            <div>content</div>
          </AppShell>
        </MemoryRouter>
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Incidents' }));
    expect(onNavigate).toHaveBeenCalledWith('incidents');
  });

  it('shows signed-in user and sign out button', () => {
    const onSignOut = vi.fn().mockResolvedValue(undefined);
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
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
          </AppShell>
        </MemoryRouter>
      </I18nextProvider>,
    );

    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeInTheDocument();
  });

  it('calls sign out and navigates to login', async () => {
    const onSignOut = vi.fn().mockResolvedValue(undefined);
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter initialEntries={['/shifts']}>
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
          </AppShell>
        </MemoryRouter>
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Sign out' }));

    await waitFor(() => {
      expect(onSignOut).toHaveBeenCalled();
    });
  });
});
