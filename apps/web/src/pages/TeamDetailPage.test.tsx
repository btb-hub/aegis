import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { TeamDetailPage } from './TeamDetailPage';

const team = {
  id: 'team-1',
  name: 'Platform',
  description: 'Core infra',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
};

const directoryUser = {
  id: 'user-2',
  email: 'bob@example.com',
  display_name: 'Bob',
  role: 'member',
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
    <MemoryRouter initialEntries={['/teams/team-1']}>
      <I18nextProvider i18n={i18n}>
        <AuthProvider>
          <Routes>
            <Route path="/teams/:teamId" element={<TeamDetailPage />} />
          </Routes>
        </AuthProvider>
      </I18nextProvider>
    </MemoryRouter>,
  );
}

describe('TeamDetailPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it('adds a member via user search picker', async () => {
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
      if (url === '/api/v1/teams/team-1/members' && init?.method === 'POST') {
        return jsonResponse(
          {
            id: 'member-2',
            team_id: 'team-1',
            user_id: 'user-2',
            team_role: 'member',
            email: 'bob@example.com',
            display_name: 'Bob',
            created_at: '2026-07-01T00:00:00Z',
          },
          201,
        );
      }
      if (url.startsWith('/api/v1/users?')) {
        return jsonResponse({ items: [directoryUser] });
      }
      if (url === '/api/v1/teams/team-1/members') {
        return jsonResponse({ items: [] });
      }
      if (url === '/api/v1/teams/team-1') {
        return jsonResponse(team);
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('No members yet')).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText('Search users'), { target: { value: 'bob' } });
    await vi.advanceTimersByTimeAsync(350);

    await waitFor(() => {
      expect(screen.getByText('Bob')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Select' }));

    await waitFor(() => {
      expect(screen.getByText('Member added')).toBeInTheDocument();
    });

    expect(fetch).toHaveBeenCalledWith('/api/v1/teams/team-1/members', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: 'user-2', team_role: 'member' }),
    });
  });

  it('renders members and updates role', async () => {
    const member = {
      id: 'member-1',
      team_id: 'team-1',
      user_id: 'user-1',
      team_role: 'member',
      email: 'alice@example.com',
      display_name: 'Alice',
      created_at: '2026-07-01T00:00:00Z',
    };

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
      if (url === '/api/v1/teams/team-1/members/user-1' && init?.method === 'PATCH') {
        return jsonResponse({ ...member, team_role: 'lead' });
      }
      if (url === '/api/v1/teams/team-1/members') {
        return jsonResponse({ items: [member] });
      }
      if (url === '/api/v1/teams/team-1') {
        return jsonResponse(team);
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Alice')).toBeInTheDocument();
    });

    const row = screen.getByText('Alice').closest('tr');
    expect(row).not.toBeNull();
    fireEvent.change(within(row as HTMLElement).getByLabelText('Team role'), { target: { value: 'lead' } });

    await waitFor(() => {
      expect(screen.getByText('Member role updated')).toBeInTheDocument();
    });
  });

  it('removes a member', async () => {
    const member = {
      id: 'member-1',
      team_id: 'team-1',
      user_id: 'user-1',
      team_role: 'member',
      email: 'alice@example.com',
      display_name: 'Alice',
      created_at: '2026-07-01T00:00:00Z',
    };

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
      if (url === '/api/v1/teams/team-1/members/user-1' && init?.method === 'DELETE') {
        return jsonResponse({}, 204);
      }
      if (url === '/api/v1/teams/team-1/members') {
        return jsonResponse({ items: [member] });
      }
      if (url === '/api/v1/teams/team-1') {
        return jsonResponse(team);
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Alice')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Remove' }));

    await waitFor(() => {
      expect(screen.getByText('Member removed')).toBeInTheDocument();
    });
  });

  it('shows load error when team fetch fails', async () => {
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
      if (url === '/api/v1/teams/team-1') {
        return jsonResponse({}, 404);
      }
      if (url === '/api/v1/teams/team-1/members') {
        return jsonResponse({ items: [] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Could not load team');
    });
  });
});
