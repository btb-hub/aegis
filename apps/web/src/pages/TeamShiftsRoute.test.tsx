import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { TeamShiftsRoute } from './TeamShiftsRoute';

function renderRoute(teamId = 'team-1') {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter initialEntries={[`/teams/${teamId}/shifts`]}>
        <AuthProvider>
          <Routes>
            <Route path="/teams/:teamId/shifts" element={<TeamShiftsRoute />} />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    </I18nextProvider>,
  );
}

describe('TeamShiftsRoute', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders on-call data from API', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return {
          ok: true,
          json: async () => ({
            id: 'user-1',
            email: 'admin@example.com',
            display_name: 'Admin',
            role: 'admin',
            locale: 'en',
            provider: 'google',
          }),
        } as Response;
      }
      if (url.match(/\/teams\/team-1$/) || url.endsWith('/teams/team-1')) {
        return {
          ok: true,
          json: async () => ({ id: 'team-1', name: 'Platform', description: '', created_at: '', updated_at: '' }),
        } as Response;
      }
      if (url.includes('/members')) {
        return {
          ok: true,
          json: async () => ({
            items: [{ id: 'm1', team_id: 'team-1', user_id: 'u1', team_role: 'member', email: 'a@x.com', display_name: 'Alice', created_at: '' }],
          }),
        } as Response;
      }
      if (url.includes('/schedules')) {
        return { ok: true, json: async () => ({ items: [{ id: 'sch-1', name: 'Primary', timezone: 'UTC', layers: [] }] }) } as Response;
      }
      if (url.includes('/on-call/current')) {
        return {
          ok: true,
          json: async () => ({
            items: [{ user_id: 'u1', email: 'a@x.com', display_name: 'Alice', source: 'rotation' }],
          }),
        } as Response;
      }
      if (url.includes('/on-call/calendar')) {
        return {
          ok: true,
          json: async () => ({
            items: [
              {
                id: 'slot-1',
                team_id: 'team-1',
                user_id: 'u1',
                start_at: '2026-06-01T00:00:00Z',
                end_at: '2026-06-08T00:00:00Z',
                source: 'rotation',
              },
            ],
          }),
        } as Response;
      }
      if (url.includes('/overrides')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      return { ok: false, status: 500, json: async () => ({}) } as Response;
    });

    renderRoute();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Platform' })).toBeInTheDocument();
    });
    expect(screen.getByText(/on call now/i)).toBeInTheDocument();
    expect(screen.getAllByText('Alice').length).toBeGreaterThan(0);
  });

  it('shows empty state when team has no schedule', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return {
          ok: true,
          json: async () => ({
            id: 'user-1',
            email: 'admin@example.com',
            display_name: 'Admin',
            role: 'member',
            locale: 'en',
            provider: 'google',
          }),
        } as Response;
      }
      if (url.match(/\/teams\/team-1$/) || url.endsWith('/teams/team-1')) {
        return {
          ok: true,
          json: async () => ({ id: 'team-1', name: 'Platform', description: '', created_at: '', updated_at: '' }),
        } as Response;
      }
      if (url.includes('/members')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      if (url.includes('/schedules')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      if (url.includes('/on-call/current')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      if (url.includes('/overrides')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      return { ok: false, status: 500, json: async () => ({}) } as Response;
    });

    renderRoute();

    await waitFor(() => {
      expect(screen.getByText(/no on-call schedule yet/i)).toBeInTheDocument();
    });
  });

  it('shows admin schedule actions when signed in as admin', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return {
          ok: true,
          json: async () => ({
            id: 'user-1',
            email: 'admin@example.com',
            display_name: 'Admin',
            role: 'admin',
            locale: 'en',
            provider: 'google',
          }),
        } as Response;
      }
      if (url.match(/\/teams\/team-1$/) || url.endsWith('/teams/team-1')) {
        return {
          ok: true,
          json: async () => ({ id: 'team-1', name: 'Platform', description: '', created_at: '', updated_at: '' }),
        } as Response;
      }
      if (url.includes('/members')) {
        return {
          ok: true,
          json: async () => ({
            items: [{ id: 'm1', team_id: 'team-1', user_id: 'u1', team_role: 'member', email: 'a@x.com', display_name: 'Alice', created_at: '' }],
          }),
        } as Response;
      }
      if (url.includes('/schedules')) {
        return {
          ok: true,
          json: async () => ({
            items: [{ id: 'sch-1', name: 'Primary', timezone: 'UTC', layers: [{ handoff_weekday: 1, handoff_time: '09:00', participant_user_ids: ['u1'] }] }],
          }),
        } as Response;
      }
      if (url.includes('/on-call/current')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      if (url.includes('/on-call/calendar')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      if (url.includes('/overrides') && init?.method === 'POST') {
        return {
          ok: true,
          json: async () => ({
            id: 'o1',
            team_id: 'team-1',
            user_id: 'u1',
            start_at: '2026-06-10T08:00:00Z',
            end_at: '2026-06-10T16:00:00Z',
          }),
        } as Response;
      }
      if (url.includes('/overrides')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      return { ok: false, status: 500, json: async () => ({}) } as Response;
    });

    renderRoute();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Edit schedule' })).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: 'Add override' }));
    expect(screen.getByRole('dialog')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Start'), { target: { value: '2026-06-10T08:00' } });
    fireEvent.change(screen.getByLabelText('End'), { target: { value: '2026-06-10T16:00' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save override' }));

    await waitFor(() => {
      expect(screen.getByText('Override saved')).toBeInTheDocument();
    });
  });

  it('shows retry when load fails', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return {
          ok: true,
          json: async () => ({
            id: 'user-1',
            email: 'admin@example.com',
            display_name: 'Admin',
            role: 'member',
            locale: 'en',
            provider: 'google',
          }),
        } as Response;
      }
      return { ok: false, status: 500, json: async () => ({}) } as Response;
    });

    renderRoute();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
    });
  });

  it('creates schedule from empty state as admin', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return {
          ok: true,
          json: async () => ({
            id: 'user-1',
            email: 'admin@example.com',
            display_name: 'Admin',
            role: 'admin',
            locale: 'en',
            provider: 'google',
          }),
        } as Response;
      }
      if (url.match(/\/teams\/team-1$/) || url.endsWith('/teams/team-1')) {
        return {
          ok: true,
          json: async () => ({ id: 'team-1', name: 'Platform', description: '', created_at: '', updated_at: '' }),
        } as Response;
      }
      if (url.includes('/members')) {
        return {
          ok: true,
          json: async () => ({
            items: [{ id: 'm1', team_id: 'team-1', user_id: 'u1', team_role: 'member', email: 'a@x.com', display_name: 'Alice', created_at: '' }],
          }),
        } as Response;
      }
      if (url.includes('/schedules') && init?.method === 'POST') {
        return {
          ok: true,
          json: async () => ({
            id: 'sch-1',
            name: 'Primary',
            timezone: 'UTC',
            layers: [{ handoff_weekday: 1, handoff_time: '09:00', participant_user_ids: ['u1'] }],
          }),
        } as Response;
      }
      if (url.includes('/schedules')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      if (url.includes('/on-call/current')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      if (url.includes('/on-call/calendar')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      if (url.includes('/overrides')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      return { ok: false, status: 500, json: async () => ({}) } as Response;
    });

    renderRoute();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Create schedule' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Create schedule' }));
    fireEvent.click(screen.getAllByRole('checkbox')[0]);
    fireEvent.click(screen.getByRole('button', { name: 'Save schedule' }));

    await waitFor(() => {
      expect(screen.getByText('Schedule saved')).toBeInTheDocument();
    });
  });

  it('updates existing schedule as admin', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return {
          ok: true,
          json: async () => ({
            id: 'user-1',
            email: 'admin@example.com',
            display_name: 'Admin',
            role: 'admin',
            locale: 'en',
            provider: 'google',
          }),
        } as Response;
      }
      if (url.match(/\/teams\/team-1$/) || url.endsWith('/teams/team-1')) {
        return {
          ok: true,
          json: async () => ({ id: 'team-1', name: 'Platform', description: '', created_at: '', updated_at: '' }),
        } as Response;
      }
      if (url.includes('/members')) {
        return {
          ok: true,
          json: async () => ({
            items: [{ id: 'm1', team_id: 'team-1', user_id: 'u1', team_role: 'member', email: 'a@x.com', display_name: 'Alice', created_at: '' }],
          }),
        } as Response;
      }
      if (url.includes('/schedules') && init?.method === 'PATCH') {
        return {
          ok: true,
          json: async () => ({
            id: 'sch-1',
            name: 'Primary',
            timezone: 'UTC',
            layers: [{ handoff_weekday: 1, handoff_time: '09:00', participant_user_ids: ['u1'] }],
          }),
        } as Response;
      }
      if (url.includes('/schedules')) {
        return {
          ok: true,
          json: async () => ({
            items: [{ id: 'sch-1', name: 'Primary', timezone: 'UTC', layers: [{ handoff_weekday: 1, handoff_time: '09:00', participant_user_ids: ['u1'] }] }],
          }),
        } as Response;
      }
      if (url.includes('/on-call/current')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      if (url.includes('/on-call/calendar')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      if (url.includes('/overrides')) {
        return { ok: true, json: async () => ({ items: [] }) } as Response;
      }
      return { ok: false, status: 500, json: async () => ({}) } as Response;
    });

    renderRoute();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Edit schedule' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Edit schedule' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save schedule' }));

    await waitFor(() => {
      expect(screen.getByText('Schedule saved')).toBeInTheDocument();
    });
  });
});
