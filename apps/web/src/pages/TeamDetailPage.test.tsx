import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { TeamDetailPage } from './TeamDetailPage';

const team = {
  id: 'team-1',
  workspace_id: '00000000-0000-0000-0000-000000000001',
  name: 'Platform',
  description: 'Core infra',
  support_tier: 'l2',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
};

function mockTeamDetailFetch(url: string, init?: RequestInit, extra?: Record<string, unknown>) {
  if (url.includes('/escalation-paths/outgoing') || url.includes('/escalation-paths/incoming')) {
    return jsonResponse({ items: [] });
  }
  if (url === '/api/v1/workspaces') {
    return jsonResponse({
      items: [
        {
          id: '00000000-0000-0000-0000-000000000001',
          name: 'Default',
          slug: 'default',
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
    return jsonResponse({ items: [team] });
  }
  if (url === '/api/v1/teams/team-1/members' && init?.method === 'POST') {
    return jsonResponse(extra?.createdMember ?? {}, 201);
  }
  if (url.startsWith('/api/v1/users?')) {
    return jsonResponse({ items: [directoryUser] });
  }
  if (url === '/api/v1/teams/team-1/members') {
    return jsonResponse({ items: extra?.members ?? [] });
  }
  if (url === '/api/v1/teams/team-1') {
    return jsonResponse(extra?.team ?? team);
  }
  return null;
}

const directoryUser = {
  id: 'user-2',
  email: 'bob@example.com',
  display_name: 'Bob',
  role: 'member',
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
    <MemoryRouter initialEntries={['/teams/team-1']}>
      <I18nextProvider i18n={i18n}>
        <AuthProvider>
          <Routes>
            <Route path="/teams/:teamId" element={<TeamDetailPage />} />
          </Routes>
        </AuthProvider>
      </I18nextProvider>
    </MemoryRouter>,
  );
}

describe('TeamDetailPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it('adds a member via user search picker', async () => {
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
      const mocked = mockTeamDetailFetch(url, init, {
        createdMember: {
          id: 'member-2',
          team_id: 'team-1',
          user_id: 'user-2',
          team_role: 'member',
          email: 'bob@example.com',
          display_name: 'Bob',
          created_at: '2026-07-01T00:00:00Z',
        },
      });
      if (mocked) {
        return mocked;
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('No members yet')).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText('Search users'), { target: { value: 'bob' } });
    await vi.advanceTimersByTimeAsync(350);

    await waitFor(() => {
      expect(screen.getByText('Bob')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Select' }));

    await waitFor(() => {
      expect(screen.getByText('Member added')).toBeInTheDocument();
    });

    expect(fetch).toHaveBeenCalledWith('/api/v1/teams/team-1/members', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: 'user-2', team_role: 'member' }),
    });
  });

  it('renders members and updates role', async () => {
    const member = {
      id: 'member-1',
      team_id: 'team-1',
      user_id: 'user-1',
      team_role: 'member',
      email: 'alice@example.com',
      display_name: 'Alice',
      created_at: '2026-07-01T00:00:00Z',
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
      if (url === '/api/v1/teams/team-1/members/user-1' && init?.method === 'PATCH') {
        return jsonResponse({ ...member, team_role: 'lead' });
      }
      const mocked = mockTeamDetailFetch(url, init, { members: [member] });
      if (mocked) {
        return mocked;
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Alice')).toBeInTheDocument();
    });

    const row = screen.getByText('Alice').closest('tr');
    expect(row).not.toBeNull();
    fireEvent.change(within(row as HTMLElement).getAllByLabelText('Team role')[0], {
      target: { value: 'lead' },
    });

    await waitFor(() => {
      expect(screen.getByText('Member role updated')).toBeInTheDocument();
    });
  });

  it('removes a member', async () => {
    const member = {
      id: 'member-1',
      team_id: 'team-1',
      user_id: 'user-1',
      team_role: 'member',
      email: 'alice@example.com',
      display_name: 'Alice',
      created_at: '2026-07-01T00:00:00Z',
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
      if (url === '/api/v1/teams/team-1/members/user-1' && init?.method === 'DELETE') {
        return jsonResponse({}, 204);
      }
      const mocked = mockTeamDetailFetch(url, init, { members: [member] });
      if (mocked) {
        return mocked;
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Alice')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Remove' }));

    await waitFor(() => {
      expect(screen.getByText('Member removed')).toBeInTheDocument();
    });
  });

  it('shows load error when team fetch fails', async () => {
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
      if (url.includes('/escalation-paths/outgoing') || url.includes('/escalation-paths/incoming')) {
        return jsonResponse({ items: [] });
      }
            if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: '00000000-0000-0000-0000-000000000001',
              name: 'Default',
              slug: 'default',
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
        return jsonResponse({ items: [] });
      }
      if (url === '/api/v1/teams/team-1') {
        return jsonResponse({}, 404);
      }
      if (url === '/api/v1/teams/team-1/members') {
        return jsonResponse({ items: [] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Could not load team');
    });
  });

  it('adds an escalation path for admin users', async () => {
    const l3Team = {
      id: 'team-l3',
      workspace_id: '00000000-0000-0000-0000-000000000001',
      name: 'Platform L3',
      description: '',
      support_tier: 'l3',
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
      if (url.includes('/escalation-paths') && init?.method === 'POST') {
        return jsonResponse({
          id: 'path-1',
          from_team_id: 'team-1',
          to_team_id: 'team-l3',
          workspace_id: '00000000-0000-0000-0000-000000000001',
          cross_workspace: false,
          created_at: '2026-07-01T00:00:00Z',
        }, 201);
      }
      if (url.includes('/escalation-paths/outgoing')) {
        return jsonResponse({ items: [] });
      }
      if (url.includes('/escalation-paths/incoming')) {
        return jsonResponse({ items: [] });
      }
            if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: '00000000-0000-0000-0000-000000000001',
              name: 'Default',
              slug: 'default',
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
        return jsonResponse({ items: [team, l3Team] });
      }
      if (url === '/api/v1/teams/team-1/members') {
        return jsonResponse({ items: [] });
      }
      if (url === '/api/v1/teams/team-1') {
        return jsonResponse(team);
      }
      if (url.includes('/workspaces/')) {
        return jsonResponse({
          id: '00000000-0000-0000-0000-000000000001',
          name: 'Default',
          slug: 'default',
          description: '',
        });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Add path' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Add path' }));

    await waitFor(() => {
      expect(screen.getByText('Escalation path added')).toBeInTheDocument();
    });
  });

  it('updates support tier and removes an outgoing path', async () => {
    const l3Team = {
      id: 'team-l3',
      workspace_id: '00000000-0000-0000-0000-000000000001',
      name: 'Platform L3',
      description: '',
      support_tier: 'l3',
      created_at: '2026-07-01T00:00:00Z',
      updated_at: '2026-07-01T00:00:00Z',
    };
    const outgoingPath = {
      id: 'path-1',
      from_team_id: 'team-1',
      to_team_id: 'team-l3',
      workspace_id: '00000000-0000-0000-0000-000000000001',
      cross_workspace: false,
      created_at: '2026-07-01T00:00:00Z',
    };
    let outgoingPaths = [outgoingPath];

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
      if (url === '/api/v1/teams/team-1' && init?.method === 'PATCH') {
        return jsonResponse({ ...team, support_tier: 'l3' });
      }
      if (url === '/api/v1/escalation-paths/path-1' && init?.method === 'DELETE') {
        outgoingPaths = [];
        return jsonResponse({}, 204);
      }
      if (url.includes('/escalation-paths/outgoing')) {
        return jsonResponse({ items: outgoingPaths });
      }
      if (url.includes('/escalation-paths/incoming')) {
        return jsonResponse({ items: [] });
      }
            if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: '00000000-0000-0000-0000-000000000001',
              name: 'Default',
              slug: 'default',
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
        return jsonResponse({ items: [team, l3Team] });
      }
      if (url === '/api/v1/teams/team-1/members') {
        return jsonResponse({ items: [] });
      }
      if (url === '/api/v1/teams/team-1') {
        return jsonResponse(team);
      }
      if (url.includes('/workspaces/')) {
        return jsonResponse({
          id: '00000000-0000-0000-0000-000000000001',
          name: 'Default',
          slug: 'default',
          description: '',
        });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'Open workspace' })).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText('Support tier'), { target: { value: 'l3' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save tier' }));

    await waitFor(() => {
      expect(screen.getByText('Support tier updated')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Remove' }));

    await waitFor(() => {
      expect(screen.getByText('Escalation path removed')).toBeInTheDocument();
    });
  });

  it('adds a cross-workspace escalation path when the toggle is enabled', async () => {
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
      if (url.includes('/escalation-paths') && init?.method === 'POST') {
        posted = JSON.parse(String(init.body)) as Record<string, unknown>;
        return jsonResponse({
          id: 'path-cross',
          from_team_id: 'team-1',
          to_team_id: 'team-devops',
          workspace_id: team.workspace_id,
          cross_workspace: true,
          created_at: '2026-07-01T00:00:00Z',
        }, 201);
      }
      if (url.includes('/escalation-paths/outgoing')) {
        return jsonResponse({ items: [] });
      }
      if (url.includes('/escalation-paths/incoming')) {
        return jsonResponse({ items: [] });
      }
            if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: '00000000-0000-0000-0000-000000000001',
              name: 'Default',
              slug: 'default',
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
      if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: team.workspace_id,
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
      if (url === '/api/v1/teams/team-1/members') {
        return jsonResponse({ items: [] });
      }
      if (url === '/api/v1/teams/team-1') {
        return jsonResponse(team);
      }
      if (url.includes('/workspaces/')) {
        return jsonResponse({
          id: team.workspace_id,
          name: 'App',
          slug: 'app',
          description: '',
        });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Add path' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText('Allow team from another workspace'));
    fireEvent.change(screen.getByLabelText('To team'), { target: { value: 'team-devops' } });
    fireEvent.click(screen.getByRole('button', { name: 'Add path' }));

    await waitFor(() => {
      expect(screen.getByText('Escalation path added')).toBeInTheDocument();
    });
    expect(posted).toMatchObject({
      from_team_id: 'team-1',
      to_team_id: 'team-devops',
      cross_workspace: true,
    });
  });
});
