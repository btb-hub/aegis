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

describe('AlertsPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:alerts'),
      revokeObjectURL: vi.fn(),
    });
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo) => {
        const url = String(input);
        if (url.includes('/saved-views')) {
          return new Response(JSON.stringify({ items: [] }), { status: 200 });
        }
        if (url.includes('/export')) {
          return new Response('id,title\n1,CPU', {
            status: 200,
            headers: { 'Content-Type': 'text/csv' },
          });
        }
        return new Response(
          JSON.stringify({
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
          }),
          { status: 200 },
        );
      }),
    );
  });

  it('renders alerts table and analytics', async () => {
    renderPage();
    expect(await screen.findByText('CPU high')).toBeInTheDocument();
    expect(screen.getByText('Inline analytics')).toBeInTheDocument();
    expect(screen.getAllByText('Critical').length).toBeGreaterThan(0);
  });

  it('exports csv when export is clicked', async () => {
    renderPage();
    await screen.findByText('CPU high');
    fireEvent.click(screen.getByRole('button', { name: 'Export CSV' }));
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/api/v1/alerts/export'), {
        credentials: 'include',
      });
    });
  });
});
