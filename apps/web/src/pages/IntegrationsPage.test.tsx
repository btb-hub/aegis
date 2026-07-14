import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
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

function renderPage() {
  return render(
    <MemoryRouter>
      <I18nextProvider i18n={i18n}>
        <AuthProvider>
          <IntegrationsPage />
        </AuthProvider>
      </I18nextProvider>
    </MemoryRouter>,
  );
}

function clickTestConnection(integrationName: string) {
  const row = screen.getByText(integrationName).closest('tr');
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
      expect(screen.getByText('Slack')).toBeInTheDocument();
    });
    expect(screen.getByText('Disabled')).toBeInTheDocument();
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
      expect(screen.getByText('Slack')).toBeInTheDocument();
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
      expect(screen.getByText('Slack')).toBeInTheDocument();
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
      expect(screen.getByText('Slack')).toBeInTheDocument();
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
      expect(screen.getByText('Slack')).toBeInTheDocument();
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
    fireEvent.change(screen.getByLabelText('Jira email'), { target: { value: 'ops@example.com' } });
    fireEvent.change(screen.getByLabelText('Jira API token'), { target: { value: 'token' } });
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
      expect(screen.getByText('Slack')).toBeInTheDocument();
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
      expect(screen.getByText('Slack')).toBeInTheDocument();
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

  it('shows workspace scope badge and saves workspace integration', async () => {
    const workspaceIntegration = {
      ...jiraIntegration,
      workspace_id: '00000000-0000-0000-0000-000000000001',
      config_complete: true,
      config: { project_key: 'OPS' },
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
      if (url === '/api/v1/integrations' && init?.method === 'POST') {
        return jsonResponse(workspaceIntegration, 201);
      }
      if (url === '/api/v1/integrations') {
        return jsonResponse({ items: [slackIntegration, workspaceIntegration] });
      }
      return jsonResponse({ items: [] });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Workspace · Default')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Add integration' }));
    fireEvent.change(screen.getByLabelText('Scope'), { target: { value: '00000000-0000-0000-0000-000000000001' } });
    fireEvent.change(screen.getByLabelText('Jira project key'), { target: { value: 'OPS' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save integration' }));

    await waitFor(() => {
      expect(screen.getByText('Integration saved')).toBeInTheDocument();
    });
  });
});
