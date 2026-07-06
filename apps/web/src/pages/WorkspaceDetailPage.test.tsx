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
    expect(screen.getByRole('link', { name: 'Platform L2' })).toHaveAttribute('href', '/teams/team-1');
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
});
