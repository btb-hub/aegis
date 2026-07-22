import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { WorkspaceDetailPage } from './WorkspaceDetailPage';

const workspaceId = '00000000-0000-0000-0000-000000000001';
const team = {
  id: 'team-1',
  workspace_id: workspaceId,
  name: 'Platform L2',
  description: 'Core',
  support_tier: 'l2',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
};

const routingRule = {
  id: 'rule-1',
  workspace_id: workspaceId,
  team_id: 'team-1',
  match_labels: { team: 'platform' },
  priority: 100,
  cross_workspace: false,
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
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
    <MemoryRouter initialEntries={[`/workspaces/${workspaceId}`]}>
      <I18nextProvider i18n={i18n}>
        <AuthProvider>
          <Routes>
            <Route path="/workspaces/:workspaceId" element={<WorkspaceDetailPage />} />
          </Routes>
        </AuthProvider>
      </I18nextProvider>
    </MemoryRouter>,
  );
}

describe('WorkspaceDetailPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('loads workspace routing rules for workspace teams', async () => {
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
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: workspaceId,
              name: 'Default',
              slug: 'default',
              description: 'Default workspace',
              team_count: 1,
              routing_rule_count: 1,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team] });
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: [routingRule] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Default' })).toBeInTheDocument();
    });
    expect(screen.getByText('team=platform')).toBeInTheDocument();
    expect(screen.getAllByRole('link', { name: 'Platform L2' })[0]).toHaveAttribute('href', '/teams/team-1');
  });

  it('creates a routing rule', async () => {
    let rules: typeof routingRule[] = [routingRule];
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
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: workspaceId,
              name: 'Default',
              slug: 'default',
              description: 'Default workspace',
              team_count: 1,
              routing_rule_count: 1,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team] });
      }
      if (url === '/api/v1/routing-rules' && init?.method === 'POST') {
        const body = JSON.parse(String(init.body)) as {
          workspace_id?: string;
          team_id: string;
          match_labels: Record<string, string>;
          priority: number;
          cross_workspace?: boolean;
        };
        const created: typeof routingRule = {
          id: 'rule-2',
          workspace_id: body.workspace_id ?? workspaceId,
          team_id: body.team_id,
          match_labels: body.match_labels as { team: string },
          priority: body.priority,
          cross_workspace: Boolean(body.cross_workspace),
          created_at: '2026-07-01T00:00:00Z',
          updated_at: '2026-07-01T00:00:00Z',
        };
        rules = [...rules, created];
        return jsonResponse(created, 201);
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: rules });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add routing rule' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Add routing rule' }));
    fireEvent.change(screen.getByLabelText('Label key'), { target: { value: 'service' } });
    fireEvent.change(screen.getByLabelText('Label value'), { target: { value: 'payments' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save rule' }));

    await waitFor(() => {
      expect(screen.getByText('Routing rule created')).toBeInTheDocument();
    });
    expect(screen.getByText('service=payments')).toBeInTheDocument();
  });

  it('deletes a routing rule', async () => {
    let rules: typeof routingRule[] = [routingRule];
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
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: workspaceId,
              name: 'Default',
              slug: 'default',
              description: 'Default workspace',
              team_count: 1,
              routing_rule_count: 1,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team] });
      }
      if (url === '/api/v1/routing-rules/rule-1' && init?.method === 'DELETE') {
        rules = [];
        return jsonResponse({}, 204);
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: rules });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('team=platform')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Delete routing rule' }));
    fireEvent.click(screen.getAllByRole('button', { name: 'Delete routing rule' })[1]);

    await waitFor(() => {
      expect(screen.getByText('Routing rule deleted')).toBeInTheDocument();
    });
    expect(screen.queryByText('team=platform')).not.toBeInTheDocument();
  });

  it('edits a routing rule', async () => {
    let rules: typeof routingRule[] = [routingRule];
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
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: workspaceId,
              name: 'Default',
              slug: 'default',
              description: 'Default workspace',
              team_count: 1,
              routing_rule_count: 1,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team] });
      }
      if (url === '/api/v1/routing-rules/rule-1' && init?.method === 'PATCH') {
        const body = JSON.parse(String(init.body)) as {
          team_id: string;
          match_labels: Record<string, string>;
          priority: number;
        };
        rules = [{
          ...routingRule,
          match_labels: body.match_labels as { team: string },
          priority: body.priority,
        }];
        return jsonResponse(rules[0]);
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: rules });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('team=platform')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Edit routing rule' }));
    fireEvent.change(screen.getByLabelText('Label value'), { target: { value: 'core' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save rule' }));

    await waitFor(() => {
      expect(screen.getByText('Routing rule updated')).toBeInTheDocument();
    });
    expect(screen.getByText('team=core')).toBeInTheDocument();
  });

  it('shows load error when workspace fetch fails', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return jsonResponse({
          id: 'member-1',
          email: 'member@example.com',
          display_name: 'Member',
          role: 'member',
          locale: 'en',
          provider: 'google',
        });
      }
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({ message: 'not found' }, 404);
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Could not load workspace')).toBeInTheDocument();
    });
  });

  it('updates workspace metadata as admin', async () => {
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
      if (url === `/api/v1/workspaces/${workspaceId}` && init?.method === 'PATCH') {
        return jsonResponse({
          id: workspaceId,
          name: 'Platform Ops',
          slug: 'platform-ops',
          description: 'Updated',
        });
      }
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({ items: [] });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team] });
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: [routingRule] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Edit workspace' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Edit workspace' }));
    fireEvent.change(screen.getByLabelText('Workspace name'), { target: { value: 'Platform Ops' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save workspace' }));

    await waitFor(() => {
      expect(screen.getByText('Workspace updated')).toBeInTheDocument();
    });
    expect(screen.getByRole('heading', { name: 'Platform Ops' })).toBeInTheDocument();
  });

  it('assigns existing teams to the workspace', async () => {
    const otherTeam = {
      id: 'team-2',
      workspace_id: '00000000-0000-0000-0000-000000000002',
      name: 'Data L2',
      description: '',
      support_tier: 'l2',
      created_at: '2026-07-01T00:00:00Z',
      updated_at: '2026-07-01T00:00:00Z',
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
      if (url === `/api/v1/workspaces/${workspaceId}/teams` && init?.method === 'POST') {
        return jsonResponse({ items: [{ ...otherTeam, workspace_id: workspaceId }] });
      }
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({ items: [] });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team, otherTeam] });
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: [routingRule] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add existing teams' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Add existing teams' }));
    fireEvent.click(screen.getByRole('checkbox', { name: 'Data L2' }));
    fireEvent.click(screen.getByRole('button', { name: 'Move to workspace' }));

    await waitFor(() => {
      expect(screen.getByText('Teams moved to this workspace')).toBeInTheDocument();
    });
  });

  it('shows blocked assign error on 409', async () => {
    const otherTeam = {
      id: 'team-2',
      workspace_id: '00000000-0000-0000-0000-000000000002',
      name: 'Data L2',
      description: '',
      support_tier: 'l2',
      created_at: '2026-07-01T00:00:00Z',
      updated_at: '2026-07-01T00:00:00Z',
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
      if (url === `/api/v1/workspaces/${workspaceId}/teams` && init?.method === 'POST') {
        return jsonResponse({ message: 'conflict' }, 409);
      }
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({ items: [] });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team, otherTeam] });
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: [routingRule] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add existing teams' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Add existing teams' }));
    fireEvent.click(screen.getByRole('checkbox', { name: 'Data L2' }));
    fireEvent.click(screen.getByRole('button', { name: 'Move to workspace' }));

    await waitFor(() => {
      expect(
        screen.getByText(
          'Move blocked by escalation paths. Remove conflicting paths on team detail, then retry.',
        ),
      ).toBeInTheDocument();
    });
  });

  it('surfaces update failure and empty tier/description dashes', async () => {
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
      if (url === `/api/v1/workspaces/${workspaceId}` && init?.method === 'PATCH') {
        return jsonResponse({ message: 'slug invalid' }, 400);
      }
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({ items: [] });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({
          items: [{ ...team, support_tier: '', description: '' }],
        });
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({
          items: [{ ...routingRule, match_labels: {} }],
        });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Edit workspace' })).toBeInTheDocument();
    });

    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(1);

    fireEvent.click(screen.getByRole('button', { name: 'Edit workspace' }));
    fireEvent.change(screen.getByLabelText('Workspace name'), { target: { value: 'Nope' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save workspace' }));

    await waitFor(() => {
      expect(screen.getByText('slug invalid')).toBeInTheDocument();
    });
  });

  it('filters assign candidates and shows empty state', async () => {
    const otherTeam = {
      id: 'team-2',
      workspace_id: '00000000-0000-0000-0000-000000000002',
      name: 'Data L2',
      description: '',
      support_tier: 'l2',
      created_at: '2026-07-01T00:00:00Z',
      updated_at: '2026-07-01T00:00:00Z',
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
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({ items: [] });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team, otherTeam] });
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: [] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add existing teams' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Add existing teams' }));
    expect(screen.getByRole('checkbox', { name: 'Data L2' })).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Search teams'), { target: { value: 'zzz' } });
    await waitFor(() => {
      expect(screen.getByText('No teams in other workspaces match your search')).toBeInTheDocument();
    });
  });

  it('hides admin actions for non-admin members', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return jsonResponse({
          id: 'member-1',
          email: 'member@example.com',
          display_name: 'Member',
          role: 'member',
          locale: 'en',
          provider: 'google',
        });
      }
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
            if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: workspaceId,
              name: 'Default',
              slug: 'default',
              description: 'Default workspace',
              team_count: 1,
              routing_rule_count: 1,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team] });
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: [routingRule] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Default' })).toBeInTheDocument();
    });
    expect(screen.queryByRole('button', { name: 'Edit workspace' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Add routing rule' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Add existing teams' })).not.toBeInTheDocument();
  });

  it('shows all three connector names while integrations are loading', async () => {
    let resolveIntegrations: (value: Response) => void = () => undefined;
    const integrationsPromise = new Promise<Response>((resolve) => {
      resolveIntegrations = resolve;
    });

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
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
            if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: workspaceId,
              name: 'Default',
              slug: 'default',
              description: 'Default workspace',
              team_count: 1,
              routing_rule_count: 1,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team] });
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: [routingRule] });
      }
      if (url === '/api/v1/integrations') {
        return integrationsPromise;
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Integrations' })).toBeInTheDocument();
    });
    expect(screen.getByText('Jira')).toBeInTheDocument();
    expect(screen.getByText('Slack')).toBeInTheDocument();
    expect(screen.getByText('eXpress')).toBeInTheDocument();
    expect(screen.getByText('Loading integrations')).toBeInTheDocument();
    expect(screen.getAllByText('Inherit')).toHaveLength(3);
    expect(screen.getAllByRole('button', { name: 'Configure' })).toHaveLength(3);
    expect(screen.getAllByRole('button', { name: 'Configure' })[0]).toBeDisabled();

    resolveIntegrations(
      jsonResponse({
        items: [
          {
            id: 'jira-slot',
            workspace_id: workspaceId,
            kind: 'jira',
            name: 'Jira',
            enabled: true,
            mode: 'inherit',
            slot_status: 'using_global',
            config: { project_key: 'OPS' },
          },
        ],
      }),
    );

    await waitFor(() => {
      expect(screen.queryByText('Loading integrations')).not.toBeInTheDocument();
    });
    expect(screen.getAllByRole('button', { name: 'Configure' })[0]).not.toBeDisabled();
  });

  it('shows all workspace integration slots and only the Jira overlay in inherit mode', async () => {
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
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
            if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: workspaceId,
              name: 'Default',
              slug: 'default',
              description: 'Default workspace',
              team_count: 1,
              routing_rule_count: 1,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team] });
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: [routingRule] });
      }
      if (url === '/api/v1/integrations') {
        return jsonResponse({
          items: [
            {
              id: 'jira-slot',
              workspace_id: workspaceId,
              kind: 'jira',
              name: 'Jira',
              enabled: true,
              mode: 'inherit',
              slot_status: 'using_global',
              config: { project_key: 'OPS' },
            },
            {
              id: 'slack-slot',
              workspace_id: workspaceId,
              kind: 'slack',
              name: 'Slack',
              enabled: true,
              mode: 'inherit',
              slot_status: 'missing',
              config: {},
            },
            {
              id: 'express-slot',
              workspace_id: workspaceId,
              kind: 'express',
              name: 'eXpress',
              enabled: true,
              mode: 'inherit',
              slot_status: 'using_global',
              config: {},
            },
          ],
        });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Integrations' })).toBeInTheDocument();
    });
    expect(screen.getByText('Jira')).toBeInTheDocument();
    expect(screen.getByText('Slack')).toBeInTheDocument();
    expect(screen.getByText('eXpress')).toBeInTheDocument();
    expect(screen.getAllByText('Inherit')).toHaveLength(3);
    expect(screen.getAllByText('Using global')).toHaveLength(2);
    expect(screen.getByText('Missing — no global')).toBeInTheDocument();

    fireEvent.click(screen.getAllByRole('button', { name: 'Configure' })[0]);
    expect(screen.getByLabelText('Mode')).toHaveValue('inherit');
    expect(screen.getByLabelText('Jira project key')).toHaveValue('OPS');
    expect(screen.queryByLabelText('Jira base URL')).not.toBeInTheDocument();
  });

  it('saves a custom workspace integration with full credentials', async () => {
    let patchBody: Record<string, unknown> | undefined;
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
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'Default',
          slug: 'default',
          description: 'Default workspace',
        });
      }
            if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: workspaceId,
              name: 'Default',
              slug: 'default',
              description: 'Default workspace',
              team_count: 1,
              routing_rule_count: 1,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team] });
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: [] });
      }
      if (url === '/api/v1/integrations/jira-slot' && init?.method === 'PATCH') {
        patchBody = JSON.parse(String(init.body)) as Record<string, unknown>;
        return jsonResponse({ id: 'jira-slot' });
      }
      if (url === '/api/v1/integrations') {
        return jsonResponse({
          items: [
            {
              id: 'jira-slot',
              workspace_id: workspaceId,
              kind: 'jira',
              name: 'Jira',
              enabled: true,
              mode: 'inherit',
              slot_status: 'using_global',
              config: { project_key: 'OPS' },
            },
          ],
        });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Integrations' })).toBeInTheDocument();
    });
    fireEvent.click(screen.getAllByRole('button', { name: 'Configure' })[0]);
    const confirm = vi.spyOn(window, 'confirm').mockReturnValueOnce(false).mockReturnValueOnce(true);
    fireEvent.change(screen.getByLabelText('Mode'), { target: { value: 'custom' } });
    fireEvent.change(screen.getByLabelText('Jira project key'), { target: { value: 'OPS' } });
    fireEvent.change(screen.getByLabelText('Mode'), { target: { value: 'inherit' } });
    expect(screen.getByLabelText('Mode')).toHaveValue('custom');
    fireEvent.change(screen.getByLabelText('Mode'), { target: { value: 'inherit' } });
    expect(screen.getByLabelText('Mode')).toHaveValue('inherit');
    expect(confirm).toHaveBeenCalledTimes(2);
    fireEvent.change(screen.getByLabelText('Mode'), { target: { value: 'custom' } });
    fireEvent.change(screen.getByLabelText('Jira base URL'), { target: { value: 'https://jira.example.com' } });
    fireEvent.change(screen.getByLabelText('Jira email'), { target: { value: 'ops@example.com' } });
    fireEvent.change(screen.getByLabelText('Jira API token'), { target: { value: 'secret' } });
    fireEvent.change(screen.getByLabelText('Jira project key'), { target: { value: 'OPS' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save integration' }));

    await waitFor(() => {
      expect(patchBody).toEqual({
        mode: 'custom',
        enabled: true,
        config: {
          base_url: 'https://jira.example.com',
          email: 'ops@example.com',
          api_token: 'secret',
          project_key: 'OPS',
        },
      });
    });
    expect(screen.getByText('Integration saved')).toBeInTheDocument();
  });

  it('lists cross-workspace routing rules by rule workspace_id and hides foreign-owned rules', async () => {
    const otherWorkspaceId = '00000000-0000-0000-0000-000000000002';
    const devops = {
      id: 'team-devops',
      workspace_id: otherWorkspaceId,
      name: 'DevOps',
      description: '',
      support_tier: 'l3',
      created_at: '2026-07-01T00:00:00Z',
      updated_at: '2026-07-01T00:00:00Z',
    };
    const ownedCrossRule = {
      ...routingRule,
      id: 'rule-cross',
      team_id: 'team-devops',
      match_labels: { service: 'shared' },
      cross_workspace: true,
    };
    const foreignOwnedRule = {
      ...routingRule,
      id: 'rule-other-ws',
      workspace_id: otherWorkspaceId,
      team_id: 'team-1',
      match_labels: { service: 'orphan' },
      cross_workspace: false,
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
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'App',
          slug: 'app',
          description: '',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: workspaceId,
              name: 'App',
              slug: 'app',
              description: '',
              team_count: 1,
              routing_rule_count: 1,
              created_at: '',
              updated_at: '',
            },
            {
              id: otherWorkspaceId,
              name: 'Ops',
              slug: 'ops',
              description: '',
              team_count: 1,
              routing_rule_count: 1,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team, devops] });
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: [ownedCrossRule, foreignOwnedRule] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('service=shared')).toBeInTheDocument();
    });
    expect(screen.getByRole('link', { name: 'DevOps (Ops)' })).toBeInTheDocument();
    expect(screen.queryByText('service=orphan')).not.toBeInTheDocument();
  });

  it('creates a cross-workspace routing rule when the toggle is enabled', async () => {
    const otherWorkspaceId = '00000000-0000-0000-0000-000000000002';
    const devops = {
      id: 'team-devops',
      workspace_id: otherWorkspaceId,
      name: 'DevOps',
      description: '',
      support_tier: 'l3',
      created_at: '2026-07-01T00:00:00Z',
      updated_at: '2026-07-01T00:00:00Z',
    };
    let rules: typeof routingRule[] = [];
    let posted: Record<string, unknown> | null = null;

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
      if (url === `/api/v1/workspaces/${workspaceId}`) {
        return jsonResponse({
          id: workspaceId,
          name: 'App',
          slug: 'app',
          description: '',
        });
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: workspaceId,
              name: 'App',
              slug: 'app',
              description: '',
              team_count: 1,
              routing_rule_count: 0,
              created_at: '',
              updated_at: '',
            },
            {
              id: otherWorkspaceId,
              name: 'Ops',
              slug: 'ops',
              description: '',
              team_count: 1,
              routing_rule_count: 0,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [team, devops] });
      }
      if (url === '/api/v1/routing-rules' && init?.method === 'POST') {
        posted = JSON.parse(String(init.body)) as Record<string, unknown>;
        const created = {
          id: 'rule-cross',
          workspace_id: workspaceId,
          team_id: String(posted.team_id),
          match_labels: posted.match_labels as { team: string },
          priority: Number(posted.priority),
          cross_workspace: Boolean(posted.cross_workspace),
          created_at: '2026-07-01T00:00:00Z',
          updated_at: '2026-07-01T00:00:00Z',
        };
        rules = [created];
        return jsonResponse(created, 201);
      }
      if (url === '/api/v1/routing-rules') {
        return jsonResponse({ items: rules });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add routing rule' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Add routing rule' }));
    fireEvent.click(screen.getByLabelText('Allow team from another workspace'));
    fireEvent.change(screen.getByLabelText('Target team'), { target: { value: 'team-devops' } });
    fireEvent.change(screen.getByLabelText('Label key'), { target: { value: 'service' } });
    fireEvent.change(screen.getByLabelText('Label value'), { target: { value: 'shared' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save rule' }));

    await waitFor(() => {
      expect(screen.getByText('Routing rule created')).toBeInTheDocument();
    });
    expect(posted).toMatchObject({
      workspace_id: workspaceId,
      team_id: 'team-devops',
      cross_workspace: true,
      match_labels: { service: 'shared' },
    });
    expect(screen.getByText('service=shared')).toBeInTheDocument();
  });
});
