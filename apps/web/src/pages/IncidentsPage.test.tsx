import { render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { IncidentsPage } from './IncidentsPage';
import i18n from '../i18n';

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}

describe('IncidentsPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders list and detail together', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/api/v1/incidents/11111111-1111-1111-1111-111111111111/timeline')) {
        return jsonResponse({ items: [] });
      }
      if (url.includes('/api/v1/incidents/11111111-1111-1111-1111-111111111111')) {
        return jsonResponse({
          incident: {
            id: '11111111-1111-1111-1111-111111111111',
            team_id: 'team-1',
            status: 'open',
            severity: 'critical',
            title: 'CPU high',
            fingerprint: 'fp-1',
            created_at: '2026-06-26T10:00:00Z',
          },
          alerts: [],
        });
      }
      if (url.includes('/handoff-targets')) {
        return jsonResponse({ items: [{ id: 'team-l3', name: 'L3' }] });
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
      if (url.includes('/api/v1/incidents')) {
        return jsonResponse({
          items: [
            {
              id: '11111111-1111-1111-1111-111111111111',
              team_id: 'team-1',
              status: 'open',
              severity: 'critical',
              title: 'CPU high',
              fingerprint: 'fp-1',
              created_at: '2026-06-26T10:00:00Z',
            },
          ],
        });
      }
      return jsonResponse({}, 404);
    });

    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <IncidentsPage />
        </MemoryRouter>
      </I18nextProvider>,
    );

    expect(screen.getByRole('heading', { name: 'Incidents' })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getAllByText('CPU high').length).toBeGreaterThan(0);
    });
  });

  it('shows a prompt when no incident is selected', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ items: [] }));

    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <IncidentsPage />
        </MemoryRouter>
      </I18nextProvider>,
    );

    await waitFor(() => {
      expect(screen.getByText('Select an incident to view details')).toBeInTheDocument();
    });
  });
});
