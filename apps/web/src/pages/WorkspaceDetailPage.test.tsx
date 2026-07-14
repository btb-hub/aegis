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
  team_id: 'team-1',
  match_labels: { team: 'platform' },
  priority: 100,
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
          team_id: string;
          match_labels: Record<string, string>;
          priority: number;
        };
        const created: typeof routingRule = {
          id: 'rule-2',
          team_id: body.team_id,
          match_labels: body.match_labels as { team: string },
          priority: body.priority,
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
});
