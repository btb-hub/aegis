import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import i18n from '../i18n';
import { DashboardPage } from './DashboardPage';

const overviewPayload = {
  mtta: { mean_seconds: 120, count: 2, series: [{ bucket_start: '2026-06-02T00:00:00Z', mean_seconds: 120, count: 2 }] },
  mttr: { mean_seconds: 300, count: 1, series: [] },
  noise: { items: [{ fingerprint: 'fp', title: 'CPU', count: 4 }] },
  on_call_load: { items: [{ user_id: 'u1', display_name: 'Alex', email: 'alex@example.com', page_count: 3 }] },
  handoffs: { count: 1, median_response_seconds: 90 },
  escalation: {
    total_incidents: 5,
    escalated_count: 1,
    escalated_percent: 20,
    mean_seconds_to_escalate: 600,
  },
};

function renderDashboard() {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>
    </I18nextProvider>,
  );
}

describe('DashboardPage', () => {
  it('renders overview widgets', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => overviewPayload,
      }),
    );

    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('Mean time to acknowledge')).toBeInTheDocument();
      expect(screen.getByText('CPU')).toBeInTheDocument();
    });
  });

  it('shows load error when overview request fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
      }),
    );

    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <DashboardPage />
        </MemoryRouter>
      </I18nextProvider>,
    );

    await waitFor(() => {
      expect(screen.getByText('Could not load analytics')).toBeInTheDocument();
    });
  });

  it('reloads when compare previous is toggled', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue({
        ok: true,
        json: async () => ({
          mtta: { mean_seconds: 120, count: 2, series: [] },
          mttr: { mean_seconds: 300, count: 1, series: [] },
          noise: { items: [] },
          on_call_load: { items: [] },
          handoffs: { count: 0, median_response_seconds: 0 },
          escalation: {
            total_incidents: 0,
            escalated_count: 0,
            escalated_percent: 0,
            mean_seconds_to_escalate: 0,
          },
        }),
      });
    vi.stubGlobal('fetch', fetchMock);

    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <DashboardPage />
        </MemoryRouter>
      </I18nextProvider>,
    );

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole('checkbox'));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  });

  it('supports drill-down navigation actions', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => overviewPayload,
      }),
    );

    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('CPU')).toBeInTheDocument();
    });

    fireEvent.click(screen.getAllByRole('button', { name: 'View incidents' })[0]);
    fireEvent.click(screen.getByRole('button', { name: 'View alerts' }));
    fireEvent.click(screen.getByRole('button', { name: 'CPU' }));
  });

  it('shows unauthorized message', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
      }),
    );

    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('Your session expired. Sign in again.')).toBeInTheDocument();
    });
  });
});
