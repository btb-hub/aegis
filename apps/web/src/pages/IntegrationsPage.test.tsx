import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { IntegrationsPage } from './IntegrationsPage';

const slackIntegration = {
  id: 'int-slack',
  kind: 'slack',
  name: 'Slack',
  enabled: true,
  config_complete: true,
  config: { bot_token: '***', signing_secret: '***' },
};

const jiraIntegration = {
  id: 'int-jira',
  kind: 'jira',
  name: 'Jira',
  enabled: false,
  config_complete: false,
  config: {},
};

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}

function LocationProbe() {
  const location = useLocation();
  return <output aria-label="Current URL">{location.pathname}{location.search}</output>;
}

function renderPage(initialEntry = '/integrations') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <I18nextProvider i18n={i18n}>
        <AuthProvider>
          <IntegrationsPage />
          <LocationProbe />
        </AuthProvider>
      </I18nextProvider>
    </MemoryRouter>,
  );
}

function clickTestConnection(integrationName: string) {
  const row = screen.getByText(integrationName, { selector: 'span' }).closest('tr');
  expect(row).not.toBeNull();
  fireEvent.click(within(row as HTMLElement).getByRole('button', { name: 'Test connection' }));
}

function authAdmin(input: RequestInfo | URL) {
  const url = String(input);
  if (url.includes('/auth/me')) {
    return jsonResponse({
      id: 'admin-1',
      email: 'admin@example.com',
      display_name: 'Admin',
      role: 'admin',
      locale: 'en',
      provider: 'google',
    });
  }
  if (url === '/api/v1/workspaces') {
    return jsonResponse({ items: [] });
  }
  return null;
}

describe('IntegrationsPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  function mockFetch(body: unknown, status = 200) {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const auth = authAdmin(input);
      if (auth) {
        return auth;
      }
      return jsonResponse(body, status);
    });
  }

  it('opens and saves Slack credentials directly when no global bot exists', async () => {
    mockFetch({ items: [] });
    renderPage();
    expect(screen.queryByRole('button', { name: 'Configure Slack' })).not.toBeInTheDocument();
    fireEvent.click(await screen.findByRole('button', { name: 'Configure Slack' }));
    const dialog = within(screen.getByRole('dialog'));
    expect(dialog.getByLabelText(/^Slack bot token/)).toHaveValue('');
    expect(dialog.getByLabelText(/^Slack signing secret/)).toHaveValue('');
    expect(dialog.getByRole('button', { name: 'Save integration' })).toBeDisabled();
    expect(dialog.getByRole('textbox', { name: 'Slack interactivity request URL' })).toHaveValue(
      `${window.location.origin}/api/v1/callbacks/slack/interactive`,
    );
    expect(dialog.getByRole('textbox', { name: 'Slack interactivity request URL' })).toHaveAttribute('readonly');
    fireEvent.change(dialog.getByLabelText(/^Slack bot token/), { target: { value: 'test-bot-token' } });
    expect(dialog.getByRole('button', { name: 'Save integration' })).toBeDisabled();
    fireEvent.change(dialog.getByLabelText(/^Slack signing secret/), { target: { value: 'test-signing-secret' } });
    fireEvent.click(dialog.getByRole('button', { name: 'Save integration' }));
    await screen.findByText('Integration saved');
    const post = vi.mocked(fetch).mock.calls.find(([url, init]) =>
      String(url) === '/api/v1/integrations' && init?.method === 'POST',
    );
    expect(JSON.parse(String(post?.[1]?.body))).toMatchObject({
      kind: 'slack', enabled: true,
      config: { bot_token: 'test-bot-token', signing_secret: 'test-signing-secret' },
    });
  });

  it('offers global Slack setup when only workspace slots exist', async () => {
    mockFetch({ items: [{ ...slackIntegration, id: 'slot-slack', workspace_id: 'workspace-1', mode: 'inherit' }] });
    renderPage();
    fireEvent.click(await screen.findByRole('button', { name: 'Configure Slack' }));
    expect(screen.getByRole('dialog')).toHaveTextContent('Add integration');
    expect(screen.getByLabelText(/^Slack bot token/)).toBeInTheDocument();
  });

  it.each([
    [true, true, 'Configured'],
    [true, false, 'Missing credentials'],
    [false, true, 'Disabled'],
  ])('shows global Slack setup status and edits the existing bot (%s, %s)', async (enabled, complete, status) => {
    mockFetch({ items: [{ ...slackIntegration, enabled, config_complete: complete }] });
    renderPage();
    const action = await screen.findByRole('button', { name: 'Configure Slack' });
    expect(screen.getByRole('region', { name: 'Slack bot' })).toHaveTextContent(status);
    fireEvent.click(action);
    const dialog = within(screen.getByRole('dialog'));
    expect(dialog.getByRole('combobox', { name: 'Kind' })).toBeDisabled();
    expect(dialog.getByLabelText(/^Slack bot token/)).toHaveValue('');
    expect(dialog.getByLabelText(/^Slack signing secret/)).toHaveValue('');
  });

  it.each(['member', 'viewer'])('explains admin access and hides credential actions for %s', async (role) => {
    vi.mocked(fetch).mockImplementation(async (input) => {
      if (String(input) === '/auth/me') {
        return jsonResponse({ id: 'user-1', role, locale: 'en' });
      }
      return String(input) === '/api/v1/workspaces'
        ? jsonResponse({ items: [] })
        : jsonResponse({ items: [slackIntegration] });
    });
    renderPage();
    await screen.findByText('Ask an administrator to configure Slack.');
    expect(screen.queryByRole('button', { name: 'Configure Slack' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Configure' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Test connection' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Add integration' })).not.toBeInTheDocument();
  });

  it('does not offer Slack setup when inventory failed to load', async () => {
    mockFetch({}, 500);
    renderPage();
    await screen.findByRole('alert');
    expect(screen.queryByRole('button', { name: 'Configure Slack' })).not.toBeInTheDocument();
  });

  it.each([false, true])('opens the Slack deep link once (existing bot: %s)', async (exists) => {
    mockFetch({ items: exists ? [slackIntegration] : [] });
    renderPage('/integrations?configure=slack&keep=value');
    const dialog = within(await screen.findByRole('dialog'));
    expect(dialog.getByLabelText(/^Slack bot token/)).toBeInTheDocument();
    expect(dialog.getByRole('combobox', { name: 'Kind' })).toHaveProperty('disabled', exists);
    await waitFor(() => expect(screen.getByLabelText('Current URL')).toHaveTextContent('/integrations?keep=value'));
    fireEvent.click(dialog.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    if (exists) {
      fireEvent.click(screen.getByRole('button', { name: 'Disable' }));
      await screen.findByText('Integration disabled');
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    }
  });

  it('waits for admin identity before opening the Slack deep link', async () => {
    let resolveAuth!: (response: Response) => void;
    const auth = new Promise<Response>((resolve) => { resolveAuth = resolve; });
    vi.mocked(fetch).mockImplementation(async (input) =>
      String(input) === '/auth/me' ? auth : jsonResponse({ items: [] }),
    );
    renderPage('/integrations?configure=slack');
    await screen.findByText('No integrations yet. Add Jira, Slack, or eXpress with credentials on this page.');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    resolveAuth(authAdmin('/auth/me') as Response);
    expect(await screen.findByRole('dialog')).toHaveTextContent('Slack bot token');
  });

  it('keeps a failed Slack deep link closed', async () => {
    mockFetch({}, 500);
    renderPage('/integrations?configure=slack');
    await screen.findByRole('alert');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('does not open a Slack deep link for a member', async () => {
    vi.mocked(fetch).mockImplementation(async (input) => String(input) === '/auth/me'
      ? jsonResponse({ id: 'member-1', role: 'member', locale: 'en' })
      : jsonResponse({ items: [] }),
    );
    renderPage('/integrations?configure=slack');
    await screen.findByText('Ask an administrator to configure Slack.');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('shows breadcrumb navigation back to shifts', async () => {
    mockFetch({ items: [] });

    renderPage();

    expect(screen.getByRole('link', { name: 'Platform' })).toHaveAttribute('href', '/dashboard');
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Integrations');
    expect(screen.getByRole('heading', { name: 'Integrations', level: 1 })).toBeInTheDocument();
  });

  it('shows loading then empty state', async () => {
    mockFetch({ items: [] });

    renderPage();
    expect(screen.getByText('Loading integrations')).toBeInTheDocument();

    await waitFor(() => {
      expect(
        screen.getByText('No integrations yet. Add Jira, Slack, or eXpress with credentials on this page.'),
      ).toBeInTheDocument();
    });
  });

  it('renders integrations and tests connection successfully', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const auth = authAdmin(input);
      if (auth) {
        return auth;
      }
      const url = String(input);
      if (url.includes('/test') && init?.method === 'POST') {
        return jsonResponse({});
      }
      if (url.includes('/integrations')) {
        return jsonResponse({ items: [slackIntegration, jiraIntegration] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack', { selector: 'span' })).toBeInTheDocument();
    });
    expect(screen.getByText('Disabled', { selector: 'td' })).toBeInTheDocument();
    expect(screen.getByText('Add credentials to finish setup')).toBeInTheDocument();

    clickTestConnection('Slack');

    await waitFor(() => {
      expect(screen.getByText('Connection succeeded')).toBeInTheDocument();
    });
    expect(fetch).toHaveBeenLastCalledWith('/api/v1/integrations/int-slack/test', {
      method: 'POST',
      credentials: 'include',
    });
  });

  it('shows sign-in message when load returns 401', async () => {
    mockFetch({}, 401);

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Your session expired. Sign in again.');
    });
  });

  it('shows load error when fetch fails', async () => {
    mockFetch({}, 500);

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Could not load integrations');
    });
  });

  it('shows load error on network failure', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return jsonResponse({
          id: 'admin-1',
          email: 'admin@example.com',
          display_name: 'Admin',
          role: 'admin',
          locale: 'en',
          provider: 'google',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({ items: [] });
      }
      throw new Error('network');
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Could not load integrations');
    });
  });

  it('shows sign-in toast when test returns 401', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const auth = authAdmin(input);
      if (auth) {
        return auth;
      }
      const url = String(input);
      if (url.includes('/test') && init?.method === 'POST') {
        return jsonResponse({}, 401);
      }
      return jsonResponse({ items: [slackIntegration] });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack', { selector: 'span' })).toBeInTheDocument();
    });
    clickTestConnection('Slack');

    await waitFor(() => {
      expect(screen.getByText('Your session expired. Sign in again.')).toBeInTheDocument();
    });
  });

  it('shows API error message when test fails', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const auth = authAdmin(input);
      if (auth) {
        return auth;
      }
      const url = String(input);
      if (url.includes('/test') && init?.method === 'POST') {
        return jsonResponse({ message: 'Invalid token' }, 400);
      }
      return jsonResponse({ items: [slackIntegration] });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack', { selector: 'span' })).toBeInTheDocument();
    });
    clickTestConnection('Slack');

    await waitFor(() => {
      expect(screen.getByText('Invalid token')).toBeInTheDocument();
    });
  });

  it('shows testing label while connection test is in flight', async () => {
    let resolveTest!: (value: Response) => void;
    const testPromise = new Promise<Response>((resolve) => {
      resolveTest = resolve;
    });

    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const auth = authAdmin(input);
      if (auth) {
        return auth;
      }
      const url = String(input);
      if (url.includes('/test') && init?.method === 'POST') {
        return testPromise;
      }
      return jsonResponse({ items: [slackIntegration] });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack', { selector: 'span' })).toBeInTheDocument();
    });
    clickTestConnection('Slack');

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Testing' })).toBeDisabled();
    });

    resolveTest(jsonResponse({}));
    await waitFor(() => {
      expect(screen.getByText('Connection succeeded')).toBeInTheDocument();
    });
  });

  it('shows generic test failure on unexpected rejection', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const auth = authAdmin(input);
      if (auth) {
        return auth;
      }
      const url = String(input);
      if (url.includes('/test') && init?.method === 'POST') {
        throw new Error('network');
      }
      return jsonResponse({ items: [slackIntegration] });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack', { selector: 'span' })).toBeInTheDocument();
    });
    clickTestConnection('Slack');

    await waitFor(() => {
      expect(screen.getByText('Connection failed')).toBeInTheDocument();
    });
  });

  it('requires jira credentials on global create and posts full config', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const auth = authAdmin(input);
      if (auth) {
        return auth;
      }
      const url = String(input);
      if (url === '/api/v1/integrations' && init?.method === 'POST') {
        return jsonResponse({ ...jiraIntegration, config_complete: true, enabled: true }, 201);
      }
      return jsonResponse({ items: [] });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add integration' })).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: 'Add integration' }));

    const save = screen.getByRole('button', { name: 'Save integration' });
    expect(save).toBeDisabled();

    fireEvent.change(screen.getByLabelText('Jira base URL'), { target: { value: 'https://jira.example.com' } });
    fireEvent.change(screen.getByLabelText(/^Jira email/), { target: { value: 'ops@example.com' } });
    fireEvent.change(screen.getByLabelText(/^Jira API token/), { target: { value: 'token' } });
    fireEvent.change(screen.getByLabelText('Jira project key'), { target: { value: 'OPS' } });

    await waitFor(() => {
      expect(save).not.toBeDisabled();
    });
    fireEvent.click(save);

    await waitFor(() => {
      expect(screen.getByText('Integration saved')).toBeInTheDocument();
    });

    const postCall = vi.mocked(fetch).mock.calls.find((call) => {
      const [url, init] = call;
      return String(url) === '/api/v1/integrations' && init?.method === 'POST';
    });
    expect(postCall).toBeTruthy();
    const body = JSON.parse(String(postCall?.[1]?.body));
    expect(body.config).toEqual({
      base_url: 'https://jira.example.com',
      email: 'ops@example.com',
      api_token: 'token',
      project_key: 'OPS',
      auth_type: 'bearer',
            deployment: 'server_dc',
    });
  });

  it('edit submit omits blank secrets', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const auth = authAdmin(input);
      if (auth) {
        return auth;
      }
      const url = String(input);
      if (url === '/api/v1/integrations/int-slack' && init?.method === 'PATCH') {
        return jsonResponse({ ...slackIntegration, name: 'Slack Bot' });
      }
      return jsonResponse({ items: [slackIntegration] });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack', { selector: 'span' })).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: 'Configure' }));
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Slack Bot' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save integration' }));

    await waitFor(() => {
      expect(screen.getByText('Integration saved')).toBeInTheDocument();
    });

    const patchCall = vi.mocked(fetch).mock.calls.find((call) => {
      const [url, init] = call;
      return String(url) === '/api/v1/integrations/int-slack' && init?.method === 'PATCH';
    });
    expect(patchCall).toBeTruthy();
    const body = JSON.parse(String(patchCall?.[1]?.body));
    expect(body.name).toBe('Slack Bot');
    expect(body.config).toEqual({});
  });

  it('disables and deletes an integration as admin', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const auth = authAdmin(input);
      if (auth) {
        return auth;
      }
      const url = String(input);
      if (url === '/api/v1/integrations/int-slack' && init?.method === 'PATCH') {
        return jsonResponse({ ...slackIntegration, enabled: false });
      }
      if (url === '/api/v1/integrations/int-slack' && init?.method === 'DELETE') {
        return jsonResponse(null, 204);
      }
      return jsonResponse({ items: [slackIntegration] });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack', { selector: 'span' })).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: 'Disable' }));

    await waitFor(() => {
      expect(screen.getByText('Integration disabled')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    const dialog = screen.getByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }));

    await waitFor(() => {
      expect(screen.getByText('Integration deleted')).toBeInTheDocument();
    });
  });

  it('filters the inventory by scope, kind, and status', async () => {
    const workspaceIntegration = {
      ...jiraIntegration,
      id: 'int-workspace-jira',
      workspace_id: '00000000-0000-0000-0000-000000000001',
      config_complete: true,
      enabled: true,
      mode: 'inherit',
      slot_status: 'using_global',
      config: { project_key: 'OPS' },
    };

    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return jsonResponse({
          id: 'admin-1',
          email: 'admin@example.com',
          display_name: 'Admin',
          role: 'admin',
          locale: 'en',
          provider: 'google',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [{ id: '00000000-0000-0000-0000-000000000001', name: 'Default', slug: 'default', description: '' }],
        });
      }
      if (url === '/api/v1/integrations') {
        return jsonResponse({ items: [slackIntegration, jiraIntegration, workspaceIntegration] });
      }
      return jsonResponse({ items: [] });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Workspace · Default')).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText('Scope filter'), { target: { value: 'workspace' } });
    expect(screen.getByText('Workspace · Default')).toBeInTheDocument();
    expect(screen.queryByText('Slack', { selector: 'span' })).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Kind filter'), { target: { value: 'slack' } });
    expect(screen.getByText('No integrations match these filters.')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Kind filter'), { target: { value: 'all' } });
    fireEvent.change(screen.getByLabelText('Status filter'), { target: { value: 'using_global' } });
    expect(screen.getByText('Workspace · Default')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Status filter'), { target: { value: 'disabled' } });
    expect(screen.getByText('No integrations match these filters.')).toBeInTheDocument();
  });

  it('configures a workspace slot in inherit mode with PATCH', async () => {
    const workspaceIntegration = {
      ...jiraIntegration,
      id: 'int-workspace-jira',
      workspace_id: '00000000-0000-0000-0000-000000000001',
      config_complete: true,
      enabled: true,
      mode: 'inherit',
      slot_status: 'using_global',
      config: { project_key: 'OLD' },
    };

    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return jsonResponse({
          id: 'admin-1',
          email: 'admin@example.com',
          display_name: 'Admin',
          role: 'admin',
          locale: 'en',
          provider: 'google',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [{ id: '00000000-0000-0000-0000-000000000001', name: 'Default', slug: 'default', description: '' }],
        });
      }
      if (url === '/api/v1/integrations/int-workspace-jira' && init?.method === 'PATCH') {
        return jsonResponse({ ...workspaceIntegration, config: { project_key: 'OPS' } });
      }
      if (url === '/api/v1/integrations') {
        return jsonResponse({ items: [workspaceIntegration] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Workspace · Default')).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: 'Configure' }));

    expect(screen.getByLabelText('Mode')).toHaveValue('inherit');
    expect(screen.queryByLabelText('Jira base URL')).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('Jira project key'), { target: { value: 'OPS' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save integration' }));

    await waitFor(() => {
      expect(screen.getByText('Integration saved')).toBeInTheDocument();
    });

    const patchCall = vi.mocked(fetch).mock.calls.find((call) => {
      const [url, init] = call;
      return String(url) === '/api/v1/integrations/int-workspace-jira' && init?.method === 'PATCH';
    });
    expect(patchCall).toBeTruthy();
    expect(JSON.parse(String(patchCall?.[1]?.body))).toEqual({
      mode: 'inherit',
      enabled: true,
      config: { project_key: 'OPS' },
    });
  });
});
