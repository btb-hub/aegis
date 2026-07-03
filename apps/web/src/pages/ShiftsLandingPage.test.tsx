import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../i18n';
import { ShiftsLandingPage } from './ShiftsLandingPage';

describe('ShiftsLandingPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('redirects to single team shifts page', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        items: [{ id: 'team-1', name: 'Platform', description: '', created_at: '', updated_at: '' }],
      }),
    } as Response);

    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter initialEntries={['/shifts']}>
          <Routes>
            <Route path="/shifts" element={<ShiftsLandingPage />} />
            <Route path="/teams/:teamId/shifts" element={<div>Shifts view</div>} />
          </Routes>
        </MemoryRouter>
      </I18nextProvider>,
    );

    await waitFor(() => {
      expect(screen.getByText('Shifts view')).toBeInTheDocument();
    });
  });

  it('lists teams when multiple exist', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        items: [
          { id: 'team-1', name: 'Platform', description: '', created_at: '', updated_at: '' },
          { id: 'team-2', name: 'Data', description: '', created_at: '', updated_at: '' },
        ],
      }),
    } as Response);

    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter initialEntries={['/shifts']}>
          <ShiftsLandingPage />
        </MemoryRouter>
      </I18nextProvider>,
    );

    expect(await screen.findByText('Platform')).toBeInTheDocument();
    expect(screen.getByText('Data')).toBeInTheDocument();
  });

  it('shows empty state when no teams exist', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ items: [] }),
    } as Response);

    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter initialEntries={['/shifts']}>
          <ShiftsLandingPage />
        </MemoryRouter>
      </I18nextProvider>,
    );

    expect(await screen.findByText(/Create a team before viewing shifts/i)).toBeInTheDocument();
  });

  it('shows error when teams fetch fails', async () => {
    vi.mocked(fetch).mockRejectedValue(new Error('network'));

    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter initialEntries={['/shifts']}>
          <ShiftsLandingPage />
        </MemoryRouter>
      </I18nextProvider>,
    );

    expect(await screen.findByText(/Could not load shifts/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
  });
});
