import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { IncidentsPage } from './IncidentsPage';
import { renderWithQuery } from '../test/renderWithQuery';

const ID_A = '11111111-1111-1111-1111-111111111111';
const ID_B = '22222222-2222-2222-2222-222222222222';

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}

function apiIncident(id: string, title: string, status = 'open') {
  return {
    id,
    team_id: 'team-1',
    status,
    severity: 'critical',
    title,
    fingerprint: `fp-${id}`,
    created_at: '2026-06-26T10:00:00Z',
  };
}

function mockIncidentsApi(options?: {
  items?: ReturnType<typeof apiIncident>[];
  timelineById?: Record<string, unknown[]>;
  detailStatusById?: Record<string, string>;
  acknowledge?: () => void;
}) {
  const items = options?.items ?? [apiIncident(ID_A, 'CPU high')];
  vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? 'GET';
    if (url.endsWith('/acknowledge') && method === 'POST') {
      options?.acknowledge?.();
      return jsonResponse({}, 204);
    }
    if (url.endsWith('/resolve') && method === 'POST') {
      return jsonResponse({}, 204);
    }
    if (url.includes('/handoff-targets')) {
      return jsonResponse({ items: [{ id: 'team-l3', name: 'L3', support_tier: 'l3' }] });
    }
    if (url.includes('/api/v1/teams/team-1')) {
      return jsonResponse({
        id: 'team-1',
        workspace_id: '00000000-0000-0000-0000-000000000001',
        name: 'Platform L2',
        description: '',
        support_tier: 'l2',
        created_at: '',
        updated_at: '',
      });
    }
    const timelineMatch = url.match(/\/api\/v1\/incidents\/([^/]+)\/timeline/);
    if (timelineMatch) {
      return jsonResponse({ items: options?.timelineById?.[timelineMatch[1]] ?? [] });
    }
    const detailMatch = url.match(/\/api\/v1\/incidents\/([^/?]+)$/);
    if (detailMatch && detailMatch[1] !== 'incidents') {
      const id = detailMatch[1];
      const listed = items.find((item) => item.id === id) ?? apiIncident(id, 'Unknown');
      return jsonResponse({
        incident: {
          ...listed,
          status: options?.detailStatusById?.[id] ?? listed.status,
        },
        alerts: [],
      });
    }
    if (url.split('?')[0] === '/api/v1/incidents') {
      return jsonResponse({ items });
    }
    return jsonResponse({}, 404);
  });
}

describe('IncidentsPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders list and detail together', async () => {
    mockIncidentsApi();
    renderWithQuery(<IncidentsPage />);

    expect(screen.getByRole('heading', { name: 'Incidents' })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getAllByText('CPU high').length).toBeGreaterThan(0);
    });
  });

  it('shows a prompt when no incident is selected', async () => {
    mockIncidentsApi({ items: [] });
    renderWithQuery(<IncidentsPage />);

    await waitFor(() => {
      expect(screen.getByText('Select an incident to view details')).toBeInTheDocument();
    });
  });

  it('keeps the list visible when selecting another incident', async () => {
    mockIncidentsApi({
      items: [apiIncident(ID_A, 'CPU high'), apiIncident(ID_B, 'Disk full')],
    });
    renderWithQuery(<IncidentsPage />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 2, name: 'CPU high' })).toBeInTheDocument();
    });

    const listCallsBefore = vi
      .mocked(fetch)
      .mock.calls.filter(([input]) => String(input).split('?')[0] === '/api/v1/incidents').length;

    fireEvent.click(screen.getByRole('button', { name: /Disk full/ }));

    expect(screen.queryByText('Loading incidents…')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /CPU high/ })).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 2, name: 'Disk full' })).toBeInTheDocument();
    });

    const listCallsAfter = vi
      .mocked(fetch)
      .mock.calls.filter(([input]) => String(input).split('?')[0] === '/api/v1/incidents').length;
    expect(listCallsAfter).toBe(listCallsBefore);
  });

  it('refreshes timeline after acknowledge without hiding the list', async () => {
    let acknowledged = false;
    mockIncidentsApi({
      acknowledge: () => {
        acknowledged = true;
      },
      get timelineById() {
        return {
          [ID_A]: acknowledged
            ? [
                {
                  id: 'evt-1',
                  kind: 'acknowledged',
                  payload: { message: 'Acknowledged' },
                  created_at: '2026-06-26T10:01:00Z',
                },
              ]
            : [],
        };
      },
      get detailStatusById() {
        return { [ID_A]: acknowledged ? 'acknowledged' : 'open' };
      },
    });
    renderWithQuery(<IncidentsPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Acknowledge' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Acknowledge' }));

    expect(screen.queryByText('Loading incidents…')).not.toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText('Acknowledged')).toBeInTheDocument();
    });
  });
});
