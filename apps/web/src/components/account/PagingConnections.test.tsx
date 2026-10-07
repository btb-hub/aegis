import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../../context/AuthContext';
import i18n from '../../i18n';
import { ProtectedRoute } from '../auth/ProtectedRoute';
import { PagingConnections } from './PagingConnections';

const initialUser = { id: 'u1', email: 'alice@example.com', display_name: 'Alice', role: 'member', locale: 'en', provider: 'google',
  slack_user_id: 'U123', express_user_huid: '00000000-0000-4000-8000-000000000001' };

describe('PagingConnections', () => {
  let user: typeof initialUser | Omit<typeof initialUser, 'slack_user_id' | 'express_user_huid'>;
  let available: boolean;
  let failure: string | null;
  let loadFails: boolean;
  const assign = vi.fn();

  beforeEach(async () => {
    user = { ...initialUser };
    available = true;
    failure = null;
    loadFails = false;
    assign.mockReset();
    vi.stubGlobal('window', new Proxy(window, { get(target, key) {
      return key === 'location' ? { assign } : Reflect.get(target, key, target);
    } }));
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url === '/auth/me') { return { ok: true, json: async () => user } as Response; }
      if (url.endsWith('/authorize')) {
        return failure
          ? { ok: false, json: async () => ({ code: `PAGING_${failure?.toUpperCase()}` }) } as Response
          : { ok: true, json: async () => ({ authorization_url: 'https://messenger.test/authorize' }) } as Response;
      }
      if (init?.method === 'DELETE') {
        if (failure) { return { ok: false, json: async () => ({ code: 'INTERNAL_ERROR' }) } as Response; }
        if (url.endsWith('/slack')) { user = { ...user }; delete (user as Partial<typeof initialUser>).slack_user_id; }
        return { ok: true, status: 204 } as Response;
      }
      return { ok: !loadFails, json: async () => ({ connections: [
        { provider: 'slack', identity: 'slack_user_id' in user ? user.slack_user_id : null, available },
        { provider: 'express', identity: 'express_user_huid' in user ? user.express_user_huid : null, available },
      ] }) } as Response;
    }));
    await i18n.changeLanguage('en');
  });
  afterEach(() => { vi.unstubAllGlobals(); });

  function show(path = '/account') {
    return render(<I18nextProvider i18n={i18n}><MemoryRouter initialEntries={[path]}><AuthProvider>
      <ProtectedRoute><PagingConnections /></ProtectedRoute>
    </AuthProvider></MemoryRouter></I18nextProvider>);
  }

  it('replaces through browser authorization and keeps the original identity', async () => {
    show();
    fireEvent.click(await screen.findByRole('button', { name: 'Replace Slack' }));
    await waitFor(() => expect(assign).toHaveBeenCalledWith('https://messenger.test/authorize'));
    expect(fetch).toHaveBeenCalledWith('/api/v1/users/me/paging-connections/slack/authorize', expect.objectContaining({ method: 'POST', credentials: 'include' }));
    expect(screen.getByText('U123')).toBeInTheDocument();
  });

  it('connects eXpress without changing sign-in or requiring a command', async () => {
    user = { id: 'u1', email: 'alice@example.com', display_name: 'Alice', role: 'member', locale: 'en', provider: 'google' };
    show();
    fireEvent.click(await screen.findByRole('button', { name: 'Connect eXpress' }));
    await waitFor(() => expect(assign).toHaveBeenCalled());
    expect(fetch).toHaveBeenCalledWith('/api/v1/users/me/paging-connections/express/authorize', expect.objectContaining({ method: 'POST' }));
  });

  it('confirms disconnect, refreshes the account, and stays mounted during refresh', async () => {
    show();
    fireEvent.click(await screen.findByRole('button', { name: 'Disconnect Slack' }));
    const dialog = screen.getByRole('dialog', { name: 'Disconnect Slack' });
    expect(within(dialog).getByText(/stop receiving incident pages through Slack/)).toBeInTheDocument();
    expect(fetch).not.toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ method: 'DELETE' }));
    fireEvent.click(within(dialog).getByRole('button', { name: 'Disconnect' }));
    expect(await screen.findByText('Slack disconnected from paging')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Connect Slack' })).toBeInTheDocument();
    expect(screen.queryByText('U123')).not.toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith('/api/v1/users/me/paging-connections/slack', expect.objectContaining({ method: 'DELETE' }));
  });

  it('cancels disconnect without making a mutation', async () => {
    show();
    fireEvent.click(await screen.findByRole('button', { name: 'Disconnect Slack' }));
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(screen.getByText('U123')).toBeInTheDocument();
    expect(fetch).not.toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ method: 'DELETE' }));
  });

  it('allows disconnect even if administrator setup is missing', async () => {
    available = false;
    show();
    expect(await screen.findByRole('button', { name: 'Replace Slack' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Disconnect Slack' })).toBeEnabled();
    expect(screen.getAllByText(/An administrator needs to configure/)).toHaveLength(2);
  });

  it('shows an authorization error while preserving the connection', async () => {
    failure = 'workspace_mismatch';
    show();
    fireEvent.click(await screen.findByRole('button', { name: 'Replace Slack' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Choose the Slack workspace');
    expect(screen.getByText('U123')).toBeInTheDocument();
    expect(assign).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: 'Replace Slack' })).toBeEnabled();
  });

  it('shows a disconnect error without removing the connection', async () => {
    failure = 'internal_error';
    show();
    fireEvent.click(await screen.findByRole('button', { name: 'Disconnect Slack' }));
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Disconnect' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not reach the messenger');
    expect(screen.getByText('U123')).toBeInTheDocument();
  });

  it('refreshes after a successful callback and shows a paging-specific message', async () => {
    show('/account?paging_result=connected&paging_provider=slack');
    expect(await screen.findByText('Slack connected for paging')).toBeInTheDocument();
    await waitFor(() => expect(vi.mocked(fetch).mock.calls.filter(([url]) => url === '/auth/me').length).toBeGreaterThan(1));
  });

  it('shows translated cancellation and never claims a successful connection', async () => {
    await i18n.changeLanguage('ru');
    show('/account?paging_error=cancelled&paging_provider=express');
    expect(await screen.findByRole('alert')).toHaveTextContent('Авторизация отменена');
    expect(screen.queryByText('eXpress подключён для пейджинга')).not.toBeInTheDocument();
  });

  it('retries failed connection loading', async () => {
    loadFails = true;
    show();
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load messenger connections');
    loadFails = false;
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('button', { name: 'Replace Slack' })).toBeInTheDocument();
  });
});
