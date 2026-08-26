import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { AlertsPage } from './AlertsPage';

type AuthRole = 'admin' | 'member' | 'viewer';

function LocationPath() {
  const location = useLocation();
  return <div data-testid="location-path">{location.pathname}</div>;
}

function renderPage() {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter initialEntries={['/alerts']}>
        <AuthProvider>
          <LocationPath />
          <Routes>
            <Route path="/alerts" element={<AlertsPage />} />
            <Route path="/workspaces" element={<div>workspaces-list</div>} />
            <Route path="/workspaces/:workspaceId" element={<div>workspace-detail</div>} />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    </I18nextProvider>,
  );
}

function authUser(role: AuthRole) {
  return {
    id: `${role}-1`,
    email: `${role}@example.com`,
    display_name: role,
    role,
    locale: 'en',
    provider: 'google',
  };
}

function workspaceSummary(id: string, name: string) {
  return {
    id,
    name,
    slug: name.toLowerCase(),
    description: '',
    team_count: 0,
    routing_rule_count: 0,
    created_at: '',
    updated_at: '',
  };
}

const listResponse = {
  items: [
    {
      id: 'alert-1',
      fingerprint: 'fp-1',
      status: 'firing',
      severity: 'critical',
      title: 'CPU high',
      labels: { team: 'platform' },
      received_at: '2026-06-26T10:00:00Z',
      incident_id: null,
    },
  ],
  total: 1,
  page: 1,
  page_size: 25,
  analytics: {
    by_severity: { critical: 1 },
    by_status: { firing: 1 },
    top_labels: [{ key: 'team', value: 'platform', count: 1 }],
  },
};

function stubFetch(options?: {
  role?: AuthRole | null;
  workspaces?: unknown[];
  workspacesError?: boolean;
  extra?: (url: string, init?: RequestInit) => Response | undefined;
}) {
  const role = options?.role === undefined ? 'admin' : options.role;
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const extra = options?.extra?.(url, init);
      if (extra) {
        return extra;
      }
      if (url.includes('/auth/me')) {
        if (!role) {
          return new Response('{}', { status: 401 });
        }
        return new Response(JSON.stringify(authUser(role)), { status: 200 });
      }
      if (url.includes('/saved-views')) {
        return new Response(JSON.stringify({ items: [] }), { status: 200 });
      }
      if (url.includes('/api/v1/workspaces')) {
        if (options?.workspacesError) {
          return new Response(JSON.stringify({ message: 'boom' }), { status: 500 });
        }
        return new Response(JSON.stringify({ items: options?.workspaces ?? [] }), { status: 200 });
      }
      return new Response(JSON.stringify(listResponse), { status: 200 });
    }),
  );
}

describe('AlertsPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:alerts'),
      revokeObjectURL: vi.fn(),
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders alerts table and analytics', async () => {
    stubFetch();

    renderPage();
    expect(await screen.findByText('CPU high')).toBeInTheDocument();
    expect(screen.getByText('Inline analytics')).toBeInTheDocument();
  });

  it('exports csv when export is clicked', async () => {
    stubFetch({
      extra: (url) => {
        if (url.includes('/export')) {
          return new Response('id,title\n1,CPU', {
            status: 200,
            headers: { 'Content-Type': 'text/csv' },
          });
        }
        return undefined;
      },
    });

    renderPage();
    await screen.findByText('CPU high');
    fireEvent.click(screen.getByRole('button', { name: 'Export CSV' }));
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/api/v1/alerts/export'), {
        credentials: 'include',
      });
    });
  });

  it('shows sign-in error on unauthorized list', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('{}', { status: 401 })));

    renderPage();
    expect(await screen.findByText('Your session expired. Sign in again.')).toBeInTheDocument();
  });

  it('shows load error when alerts request fails', async () => {
    stubFetch({
      extra: (url) => {
        if (url.includes('/api/v1/alerts') && !url.includes('/export')) {
          return new Response('{}', { status: 500 });
        }
        return undefined;
      },
    });

    renderPage();
    expect(await screen.findByText('Could not load alerts')).toBeInTheDocument();
  });

  it('renders grouped response', async () => {
    stubFetch({
      extra: (url) => {
        if (url.includes('/api/v1/alerts') && !url.includes('/export')) {
          return new Response(
            JSON.stringify({
              group_by: 'severity',
              groups: [{ key: 'critical', count: 2, sample: listResponse.items[0] }],
              total: 2,
            }),
            { status: 200 },
          );
        }
        return undefined;
      },
    });

    renderPage();
    expect(await screen.findByText('Grouped by severity · 2 alerts')).toBeInTheDocument();
  });

  it('loads and saves a view', async () => {
    const savedView = {
      id: 'view-1',
      owner_id: 'user-1',
      name: 'Critical only',
      filter: { severity: 'critical', group_by: 'severity' },
      shared: true,
    };

    stubFetch({
      extra: (url, init) => {
        if (url.includes('/saved-views') && init?.method === 'POST') {
          return new Response(JSON.stringify(savedView), { status: 201 });
        }
        if (url.includes('/saved-views')) {
          return new Response(JSON.stringify({ items: [savedView] }), { status: 200 });
        }
        return undefined;
      },
    });

    renderPage();
    await screen.findByText('CPU high');

    fireEvent.change(screen.getByLabelText('View name'), { target: { value: 'Critical only' } });
    fireEvent.click(screen.getByLabelText('Share with team'));
    fireEvent.click(screen.getByRole('button', { name: 'Save view' }));

    expect(await screen.findByText('View saved')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Saved view'), { target: { value: 'view-1' } });
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/api/v1/alerts?'), {
        credentials: 'include',
      });
    });
  });

  it('requires a name before saving a view', async () => {
    stubFetch();

    renderPage();
    await screen.findByText('CPU high');
    fireEvent.click(screen.getByRole('button', { name: 'Save view' }));
    expect(await screen.findByText('Enter a view name')).toBeInTheDocument();
  });

  it('shows export failure toast', async () => {
    stubFetch({
      extra: (url) => {
        if (url.includes('/export')) {
          return new Response('{}', { status: 500 });
        }
        return undefined;
      },
    });

    renderPage();
    await screen.findByText('CPU high');
    fireEvent.click(screen.getByRole('button', { name: 'Export CSV' }));
    expect(await screen.findByText('Export failed')).toBeInTheDocument();
  });

  it('shows configure routing for admins without replacing export', async () => {
    stubFetch({ role: 'admin' });

    renderPage();
    await screen.findByText('CPU high');
    expect(screen.getByRole('button', { name: 'Configure routing' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Export CSV' })).toBeInTheDocument();
  });

  it('hides configure routing for members', async () => {
    stubFetch({ role: 'member' });

    renderPage();
    await screen.findByText('CPU high');
    expect(screen.queryByRole('button', { name: 'Configure routing' })).not.toBeInTheDocument();
  });

  it('hides configure routing for viewers', async () => {
    stubFetch({ role: 'viewer' });

    renderPage();
    await screen.findByText('CPU high');
    expect(screen.queryByRole('button', { name: 'Configure routing' })).not.toBeInTheDocument();
  });

  it('navigates to the workspace when there is exactly one', async () => {
    stubFetch({
      role: 'admin',
      workspaces: [workspaceSummary('ws-1', 'Default')],
    });

    renderPage();
    await screen.findByText('CPU high');
    fireEvent.click(screen.getByRole('button', { name: 'Configure routing' }));

    await waitFor(() => {
      expect(screen.getByTestId('location-path')).toHaveTextContent('/workspaces/ws-1');
    });
  });

  it('navigates to workspaces list when there are no workspaces', async () => {
    stubFetch({ role: 'admin', workspaces: [] });

    renderPage();
    await screen.findByText('CPU high');
    fireEvent.click(screen.getByRole('button', { name: 'Configure routing' }));

    await waitFor(() => {
      expect(screen.getByTestId('location-path')).toHaveTextContent('/workspaces');
    });
    expect(screen.getByTestId('location-path')).not.toHaveTextContent('/workspaces/');
  });

  it('navigates to workspaces list when there are two or more workspaces', async () => {
    stubFetch({
      role: 'admin',
      workspaces: [workspaceSummary('ws-1', 'Default'), workspaceSummary('ws-2', 'Platform')],
    });

    renderPage();
    await screen.findByText('CPU high');
    fireEvent.click(screen.getByRole('button', { name: 'Configure routing' }));

    await waitFor(() => {
      expect(screen.getByTestId('location-path')).toHaveTextContent('/workspaces');
    });
    expect(screen.queryByText('workspace-detail')).not.toBeInTheDocument();
  });

  it('toasts and stays on alerts when workspaces fail to load', async () => {
    stubFetch({ role: 'admin', workspacesError: true });

    renderPage();
    await screen.findByText('CPU high');
    fireEvent.click(screen.getByRole('button', { name: 'Configure routing' }));

    expect(await screen.findByText('Could not load workspaces')).toBeInTheDocument();
    expect(screen.getByTestId('location-path')).toHaveTextContent('/alerts');
  });
});
