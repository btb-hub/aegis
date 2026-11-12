import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
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
  let codeFails: boolean;
  let authFailure: 'server' | 'network' | 'unauthorized' | null;
  const writeText = vi.fn();
  const assign = vi.fn();

  beforeEach(async () => {
    user = { ...initialUser };
    available = true;
    failure = null;
    loadFails = false;
    codeFails = false;
    authFailure = null;
    writeText.mockReset();
    writeText.mockResolvedValue(undefined);
    vi.stubGlobal('navigator', { clipboard: { writeText } });
    assign.mockReset();
    vi.stubGlobal('window', new Proxy(window, { get(target, key) {
      return key === 'location' ? { assign } : Reflect.get(target, key, target);
    } }));
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url === '/auth/me') {
        if (authFailure === 'network') { throw new Error('network unavailable'); }
        if (authFailure) { return { ok: false, status: authFailure === 'unauthorized' ? 401 : 500 } as Response; }
        return { ok: true, json: async () => user } as Response;
      }
      if (url === '/api/v1/users/me/express-link-code') {
        return { ok: !codeFails, json: async () => codeFails
          ? { message: 'generation failed' } : { code: 'AB12CD34', command: '/link AB12CD34' } } as Response;
      }
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
      <Routes>
        <Route path="/account" element={<ProtectedRoute><PagingConnections /></ProtectedRoute>} />
        <Route path="/login" element={<p>Sign in required</p>} />
      </Routes>
    </AuthProvider></MemoryRouter></I18nextProvider>);
  }

  // EXPRESS-BOT-AC1/AC2, contract in PagingConnections.tsx. Real Account UI and
  // auth state, mocked HTTP and clipboard; this does not prove BotX delivery.
  it('generates a bot command without SSO setup and only shows linked status after refresh', async () => {
    user = { id: 'u1', email: 'alice@example.com', display_name: 'Alice', role: 'member', locale: 'en', provider: 'google' };
    available = false;
    show();
    expect(await screen.findByRole('button', { name: 'Connect eXpress' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: 'Generate link code' }));
    expect(await screen.findByLabelText('eXpress bot command')).toHaveValue('/link AB12CD34');
    expect(screen.getByText(/valid for 15 minutes/)).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith('/api/v1/users/me/express-link-code', expect.objectContaining({ method: 'POST', credentials: 'include' }));
    expect(assign).not.toHaveBeenCalled();
    expect(screen.queryByText('eXpress connected for paging')).not.toBeInTheDocument();
    expect(screen.getAllByText('Not connected')).toHaveLength(2);

    // EXPRESS-BOT-AC3: Copy exactly the server command. A successful bot command
    // is simulated by changing the HTTP fixture, then explicitly refreshing.
    fireEvent.click(screen.getByRole('button', { name: 'Copy command' }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith('/link AB12CD34'));
    user = { ...initialUser };
    fireEvent.click(screen.getByRole('button', { name: 'Check connection' }));
    expect(await screen.findByText(initialUser.express_user_huid)).toBeInTheDocument();
    expect(screen.queryByLabelText('eXpress bot command')).not.toBeInTheDocument();
  });

  // EXPRESS-BOT-AC1/AC2: A failing status request must not block code generation;
  // a failed generation leaves no command and can be retried without navigation.
  it('keeps bot linking usable when status loading fails and allows generation retry', async () => {
    loadFails = true;
    codeFails = true;
    show();
    await screen.findByText('Could not load messenger connections. Try again.');
    fireEvent.click(screen.getByRole('button', { name: 'Generate link code' }));
    expect(await screen.findByText('Could not generate link code')).toBeInTheDocument();
    expect(screen.queryByLabelText('eXpress bot command')).not.toBeInTheDocument();
    codeFails = false;
    fireEvent.click(screen.getByRole('button', { name: 'Generate link code' }));
    expect(await screen.findByLabelText('eXpress bot command')).toHaveValue('/link AB12CD34');
  });

  // EXPRESS-BOT-AC3: Clipboard denial must preserve the selectable command and
  // explain manual copying. The browser clipboard is replaced at the boundary.
  it('keeps the command available for manual copying if clipboard access fails', async () => {
    writeText.mockRejectedValue(new Error('permission denied'));
    show();
    fireEvent.click(await screen.findByRole('button', { name: 'Generate link code' }));
    await screen.findByLabelText('eXpress bot command');
    fireEvent.click(screen.getByRole('button', { name: 'Copy command' }));
    expect(await screen.findByText('Could not copy. Select and copy the command above.')).toBeInTheDocument();
    expect(screen.getByLabelText('eXpress bot command')).toHaveValue('/link AB12CD34');
  });

  // EXPRESS-BOT-AC4, AuthContext.tsx refresh: transient account failures must
  // preserve the real rendered command and allow retry. HTTP errors are mocked;
  // this checks UI/auth state, not the real session cookie or BotX callback.
  it.each(['server', 'network'] as const)('preserves the account and command after a %s failure during connection check', async (failure) => {
    show();
    fireEvent.click(await screen.findByRole('button', { name: 'Generate link code' }));
    await screen.findByLabelText('eXpress bot command');
    authFailure = failure;
    fireEvent.click(screen.getByRole('button', { name: 'Check connection' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not check your account. Your connection has not been confirmed. Try again.');
    expect(screen.getByLabelText('eXpress bot command')).toHaveValue('/link AB12CD34');
    expect(screen.queryByText('eXpress connected for paging')).not.toBeInTheDocument();
    authFailure = null;
    fireEvent.click(screen.getByRole('button', { name: 'Check connection' }));
    await waitFor(() => expect(screen.queryByRole('alert')).not.toBeInTheDocument());
    expect(screen.getByText(initialUser.express_user_huid)).toBeInTheDocument();
  });

  // EXPRESS-BOT-AC4 preserves the existing auth boundary: a genuine 401 clears
  // the protected UI even during a background check. HTTP is mocked here.
  it('returns to sign-in when the connection check confirms the session expired', async () => {
    show();
    fireEvent.click(await screen.findByRole('button', { name: 'Generate link code' }));
    await screen.findByLabelText('eXpress bot command');
    authFailure = 'unauthorized';
    fireEvent.click(screen.getByRole('button', { name: 'Check connection' }));
    expect(await screen.findByText('Sign in required')).toBeInTheDocument();
    expect(screen.queryByLabelText('eXpress bot command')).not.toBeInTheDocument();
  });

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
