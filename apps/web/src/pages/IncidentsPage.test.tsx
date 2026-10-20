import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { Link, MemoryRouter, useLocation, useNavigate } from 'react-router-dom';
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

const queueRecords = [
  { id: '11111111-1111-1111-1111-111111111111', team_id: 'team-1', status: 'open', severity: 'critical', title: 'Queue incident one', fingerprint: 'one', created_at: '2026-10-07T12:00:00Z' },
  { id: '22222222-2222-2222-2222-222222222222', team_id: 'team-1', status: 'open', severity: 'high', title: 'Queue incident two', fingerprint: 'two', created_at: '2026-10-07T13:00:00Z' },
];

function RouterControls() {
  const location = useLocation();
  const navigate = useNavigate();
  return <>
    <output aria-label="Current URL">{location.pathname}{location.search}</output>
    <button onClick={() => navigate(-1)}>Browser back</button>
    <button onClick={() => navigate(1)}>Browser forward</button>
  </>;
}

function mockQueueRequests() {
  vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
    const url = String(input);
    const path = url.split('?')[0];
    if (path.endsWith('/api/v1/incidents')) { return jsonResponse({ items: url.includes('status=resolved') ? [] : queueRecords }); }
    if (url.endsWith('/timeline') || url.endsWith('/handoff-targets')) { return jsonResponse({ items: [] }); }
    const record = queueRecords.find((item) => url.endsWith(`/incidents/${item.id}`));
    if (record) { return jsonResponse({ incident: record, alerts: [] }); }
    return jsonResponse({}, 404);
  });
}

function renderQueue(initialEntry = '/incidents') {
  return render(<I18nextProvider i18n={i18n}><MemoryRouter initialEntries={[initialEntry]}>
    <RouterControls /><IncidentsPage />
  </MemoryRouter></I18nextProvider>);
}

describe('IncidentsPage', () => {
 it('clears the previous detail when a notification link selects a missing incident', async () => {
  const id = '22222222-2222-2222-2222-222222222222';
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.endsWith(`/incidents/${id}/timeline`)) {return jsonResponse({ items: [] });}
    if (url.endsWith(`/incidents/${id}`)) {return jsonResponse({ incident: { id, team_id: 'team-link', status: 'resolved', severity: 'critical', title: 'Previous detail', fingerprint: 'linked', created_at: '2026-10-07T12:00:00Z' }, alerts: [] });}
    if (url.includes('/handoff-targets') || url.endsWith('/api/v1/incidents')) {return jsonResponse({ items: [] });}
    return jsonResponse({}, 404);
  }));
  render(<I18nextProvider i18n={i18n}><MemoryRouter initialEntries={[`/incidents?incident_id=${id}`]}><Link to="/incidents?incident_id=missing">Open missing incident</Link><IncidentsPage /></MemoryRouter></I18nextProvider>);
  await waitFor(() => expect(screen.getByText('Previous detail')).toBeInTheDocument());
  fireEvent.click(screen.getByText('Open missing incident'));
  await waitFor(() => expect(screen.queryByText('Previous detail')).not.toBeInTheDocument());
  await waitFor(() => expect(screen.getByText(i18n.t('incidents.detail_load_error'))).toBeInTheDocument());
 });
  it('opens the incident from a notification link even when the list is empty', async () => {
    const id = '22222222-2222-2222-2222-222222222222';
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith(`/incidents/${id}/timeline`)) {return jsonResponse({ items: [] });}
      if (url.endsWith(`/incidents/${id}`)) {return jsonResponse({
        incident: { id, team_id: 'team-link', status: 'resolved', severity: 'critical', title: 'Linked incident', fingerprint: 'linked', created_at: '2026-10-07T12:00:00Z' }, alerts: [],
      });}
      if (url.includes('/handoff-targets')) {return jsonResponse({ items: [] });}
      if (url.endsWith('/api/v1/incidents')) {return jsonResponse({ items: [] });}
      return jsonResponse({}, 404);
    }));
    render(<I18nextProvider i18n={i18n}><MemoryRouter initialEntries={[`/incidents?incident_id=${id}`]}><IncidentsPage /></MemoryRouter></I18nextProvider>);
    await waitFor(() => expect(screen.getAllByText('Linked incident').length).toBeGreaterThan(0));
    expect(fetch).toHaveBeenCalledWith(`/api/v1/incidents/${id}`, expect.anything());
  });
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

  it('opens a selected incident without refetching the queue and restores row focus on return', async () => {
    mockQueueRequests();
    const { container } = renderQueue('/incidents?status=open&source=notification');
    const row = await screen.findByRole('button', { name: /Queue incident two/ });
    row.focus();
    fireEvent.click(row);
    await screen.findByRole('heading', { name: 'Queue incident two' });
    expect(screen.getByLabelText('Current URL')).toHaveTextContent(`status=open&source=notification&incident_id=${queueRecords[1].id}`);
    expect(container.querySelector('[data-incident-queue]')).toHaveClass('hidden', 'xl:block');
    expect(container.querySelector('[data-incident-detail]')).not.toHaveClass('hidden');
    expect(screen.getByRole('region', { name: i18n.t('incidents.detail_label') })).toHaveFocus();
    const listCalls = vi.mocked(fetch).mock.calls.filter(([url]) => String(url).split('?')[0].endsWith('/api/v1/incidents'));
    expect(listCalls).toHaveLength(1);

    fireEvent.click(screen.getByRole('button', { name: i18n.t('incidents.back_to_list') }));
    await waitFor(() => expect(row).toHaveFocus());
    expect(row).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByLabelText('Current URL')).toHaveTextContent('/incidents?status=open&source=notification');
    expect(container.querySelector('[data-incident-queue]')).not.toHaveClass('hidden');
    expect(container.querySelector('[data-incident-detail]')).toHaveClass('hidden', 'xl:block');
  });

  it('restores selected incident and status through browser history', async () => {
    mockQueueRequests();
    renderQueue();
    fireEvent.click(await screen.findByRole('button', { name: /Queue incident two/ }));
    await screen.findByRole('heading', { name: 'Queue incident two' });
    fireEvent.click(screen.getByRole('button', { name: i18n.t('incidents.back_to_list') }));
    fireEvent.click(screen.getByRole('button', { name: i18n.t('incidents.status.resolved') }));
    await screen.findByText(i18n.t('incidents.empty_filtered'));
    expect(screen.getByLabelText('Current URL')).toHaveTextContent('/incidents?status=resolved');

    fireEvent.click(screen.getByRole('button', { name: 'Browser back' }));
    await screen.findByRole('button', { name: /Queue incident two/ });
    expect(screen.getByLabelText('Current URL')).toHaveTextContent('/incidents');
    fireEvent.click(screen.getByRole('button', { name: 'Browser back' }));
    await screen.findByRole('heading', { name: 'Queue incident two' });
    expect(screen.getByLabelText('Current URL')).toHaveTextContent(`incident_id=${queueRecords[1].id}`);
  });

  it('keeps queue rows mounted during a post-action refresh and preserves them after a refresh failure', async () => {
    let acknowledged = false;
    let listRequests = 0;
    let completeRefresh: ((response: Response) => void) | undefined;
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith('/acknowledge')) { acknowledged = true; return jsonResponse({}, 204); }
      if (url.split('?')[0].endsWith('/api/v1/incidents')) {
        listRequests += 1;
        if (listRequests > 1) { return new Promise<Response>((resolve) => { completeRefresh = resolve; }); }
        return jsonResponse({ items: queueRecords });
      }
      if (url.endsWith('/timeline') || url.endsWith('/handoff-targets')) { return jsonResponse({ items: [] }); }
      if (url.endsWith(`/incidents/${queueRecords[0].id}`)) {
        return jsonResponse({ incident: { ...queueRecords[0], status: acknowledged ? 'acknowledged' : 'open' }, alerts: [] });
      }
      return jsonResponse({}, 404);
    });
    renderQueue();
    const row = await screen.findByRole('button', { name: /Queue incident one/ });
    fireEvent.click(await screen.findByRole('button', { name: i18n.t('incidents.acknowledge') }));
    await screen.findByText(i18n.t('incidents.refreshing'));
    expect(row).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Queue incident one' })).toBeInTheDocument();
    completeRefresh?.(jsonResponse({}, 500));
    await screen.findByText(i18n.t('incidents.load_error'));
    expect(row).toBeInTheDocument();
    expect(screen.getByRole('button', { name: i18n.t('incidents.retry') })).toBeInTheDocument();
  });

  it('retries a failed detail without replacing it with the previous incident', async () => {
    mockQueueRequests();
    const successfulFetch = vi.mocked(fetch).getMockImplementation()!;
    let fail = true;
    vi.mocked(fetch).mockImplementation((input, init) => {
      if (fail && String(input).endsWith(`/incidents/${queueRecords[1].id}`)) { return Promise.resolve(jsonResponse({}, 500)); }
      return successfulFetch(input, init);
    });
    renderQueue();
    await screen.findByRole('heading', { name: 'Queue incident one' });
    fireEvent.click(screen.getByRole('button', { name: /Queue incident two/ }));
    await screen.findByText(i18n.t('incidents.detail_load_error'));
    expect(screen.queryByRole('heading', { name: 'Queue incident one' })).not.toBeInTheDocument();
    fail = false;
    fireEvent.click(screen.getByRole('button', { name: i18n.t('incidents.retry') }));
    await screen.findByRole('heading', { name: 'Queue incident two' });
  });

  it('does not reopen detail or steal row focus when an action finishes after returning to the queue', async () => {
    mockQueueRequests();
    const defaultFetch = vi.mocked(fetch).getMockImplementation()!;
    let completeMutation: ((response: Response) => void) | undefined;
    vi.mocked(fetch).mockImplementation((input, init) => {
      if (String(input).endsWith('/acknowledge')) {
        return new Promise<Response>((resolve) => { completeMutation = resolve; });
      }
      return defaultFetch(input, init);
    });
    const { container } = renderQueue();
    const row = await screen.findByRole('button', { name: /Queue incident two/ });
    row.focus();
    fireEvent.click(row);
    await screen.findByRole('heading', { name: 'Queue incident two' });
    fireEvent.click(screen.getByRole('button', { name: i18n.t('incidents.acknowledge') }));
    fireEvent.click(screen.getByRole('button', { name: i18n.t('incidents.back_to_list') }));
    expect(row).toHaveFocus();
    completeMutation?.(jsonResponse({}, 204));
    await screen.findByText(i18n.t('incidents.action_complete'));
    expect(screen.getByLabelText('Current URL').textContent).toBe('/incidents');
    expect(container.querySelector('[data-incident-detail]')).toHaveClass('hidden', 'xl:block');
    expect(container.querySelector('[data-incident-queue]')).not.toHaveClass('hidden');
    expect(row).toHaveFocus();
  });

  it('keeps the completed incident visible when resolving it removes it from the current filter', async () => {
    let resolved = false;
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith('/resolve')) { resolved = true; return jsonResponse({}, 204); }
      if (url.split('?')[0].endsWith('/api/v1/incidents')) { return jsonResponse({ items: resolved ? [] : [queueRecords[0]] }); }
      if (url.endsWith('/timeline') || url.endsWith('/handoff-targets')) { return jsonResponse({ items: [] }); }
      if (url.endsWith(`/incidents/${queueRecords[0].id}`)) {
        return jsonResponse({ incident: { ...queueRecords[0], status: resolved ? 'resolved' : 'open' }, alerts: [] });
      }
      return jsonResponse({}, 404);
    });
    renderQueue('/incidents?status=open');
    fireEvent.click(await screen.findByRole('button', { name: i18n.t('incidents.resolve') }));
    await screen.findByText(i18n.t('incidents.action_complete'));
    expect(screen.getByLabelText('Current URL')).toHaveTextContent(`status=open&incident_id=${queueRecords[0].id}`);
    expect(screen.getByRole('heading', { name: 'Queue incident one' })).toBeInTheDocument();
    expect(screen.getByText(i18n.t('incidents.empty_filtered'))).toBeInTheDocument();
  });
});
