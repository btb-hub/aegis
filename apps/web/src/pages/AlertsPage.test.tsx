import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../i18n';
import { AlertsPage } from './AlertsPage';

function renderPage() {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <AlertsPage />
      </MemoryRouter>
    </I18nextProvider>,
  );
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

describe('AlertsPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:alerts'),
      revokeObjectURL: vi.fn(),
    });
  });

  it('renders alerts table and analytics', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo) => {
        const url = String(input);
        if (url.includes('/saved-views')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return new Response(JSON.stringify(listResponse), { status: 200 });
      }),
    );

    renderPage();
    expect(await screen.findByText('CPU high')).toBeInTheDocument();
    expect(screen.getByText('Inline analytics')).toBeInTheDocument();
  });

  it('exports csv when export is clicked', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo) => {
        const url = String(input);
        if (url.includes('/export')) {
          return new Response('id,title\n1,CPU', {
            status: 200,
            headers: { 'Content-Type': 'text/csv' },
          });
        }
        if (url.includes('/saved-views')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return new Response(JSON.stringify(listResponse), { status: 200 });
      }),
    );

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
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response('{}', { status: 401 })),
    );

    renderPage();
    expect(await screen.findByText('Your session expired. Sign in again.')).toBeInTheDocument();
  });

  it('shows load error when alerts request fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo) => {
        if (String(input).includes('/saved-views')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return new Response('{}', { status: 500 });
      }),
    );

    renderPage();
    expect(await screen.findByText('Could not load alerts')).toBeInTheDocument();
  });

  it('renders grouped response', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo) => {
        if (String(input).includes('/saved-views')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return new Response(
          JSON.stringify({
            group_by: 'severity',
            groups: [{ key: 'critical', count: 2, sample: listResponse.items[0] }],
            total: 2,
          }),
          { status: 200 },
        );
      }),
    );

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

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo, init?: RequestInit) => {
        const url = String(input);
        if (url.includes('/saved-views') && init?.method === 'POST') {
          return new Response(JSON.stringify(savedView), { status: 201 });
        }
        if (url.includes('/saved-views')) {
          return new Response(JSON.stringify({ items: [savedView] }), { status: 200 });
        }
        return new Response(JSON.stringify(listResponse), { status: 200 });
      }),
    );

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
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo) => {
        if (String(input).includes('/saved-views')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return new Response(JSON.stringify(listResponse), { status: 200 });
      }),
    );

    renderPage();
    await screen.findByText('CPU high');
    fireEvent.click(screen.getByRole('button', { name: 'Save view' }));
    expect(await screen.findByText('Enter a view name')).toBeInTheDocument();
  });

  it('shows export failure toast', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo) => {
        const url = String(input);
        if (url.includes('/export')) {
          return new Response('{}', { status: 500 });
        }
        if (url.includes('/saved-views')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        return new Response(JSON.stringify(listResponse), { status: 200 });
      }),
    );

    renderPage();
    await screen.findByText('CPU high');
    fireEvent.click(screen.getByRole('button', { name: 'Export CSV' }));
    expect(await screen.findByText('Export failed')).toBeInTheDocument();
  });
});
