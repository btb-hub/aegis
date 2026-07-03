import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { TeamsPage } from './TeamsPage';

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}

function renderPage() {
  return render(
    <MemoryRouter>
      <I18nextProvider i18n={i18n}>
        <AuthProvider>
          <TeamsPage />
        </AuthProvider>
      </I18nextProvider>
    </MemoryRouter>,
  );
}

describe('TeamsPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('creates a team as admin', async () => {
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
      if (url === '/api/v1/teams' && init?.method === 'POST') {
        return jsonResponse({ id: 'team-new', name: 'Platform', description: 'Core infra' }, 201);
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('No teams yet')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Create your first team' }));
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Platform' } });
    fireEvent.change(screen.getByLabelText('Description'), { target: { value: 'Core infra' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(screen.getByText('Team created')).toBeInTheDocument();
    });

    expect(fetch).toHaveBeenCalledWith('/api/v1/teams', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'Platform', description: 'Core infra' }),
    });
  });

  it('hides create actions for non-admin users', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return jsonResponse({
          id: 'user-1',
          email: 'member@example.com',
          display_name: 'Member',
          role: 'member',
          locale: 'en',
          provider: 'google',
        });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('No teams yet')).toBeInTheDocument();
    });

    expect(screen.queryByRole('button', { name: 'Create team' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Create your first team' })).not.toBeInTheDocument();
  });

  it('lists teams and edits one as admin', async () => {
    const existingTeam = {
      id: 'team-1',
      name: 'Platform',
      description: 'Core',
      created_at: '2026-07-01T00:00:00Z',
      updated_at: '2026-07-01T00:00:00Z',
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
      if (url === '/api/v1/teams/team-1' && init?.method === 'PATCH') {
        return jsonResponse({ ...existingTeam, name: 'Platform L2' });
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [existingTeam] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'Platform' })).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Edit team' }));
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Platform L2' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(screen.getByText('Team updated')).toBeInTheDocument();
    });
  });

  it('deletes a team as admin', async () => {
    const existingTeam = {
      id: 'team-1',
      name: 'Platform',
      description: '',
      created_at: '2026-07-01T00:00:00Z',
      updated_at: '2026-07-01T00:00:00Z',
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
      if (url === '/api/v1/teams/team-1' && init?.method === 'DELETE') {
        return jsonResponse({}, 204);
      }
      if (url === '/api/v1/teams') {
        return jsonResponse({ items: [existingTeam] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Platform')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Delete team' }));
    fireEvent.click(screen.getAllByRole('button', { name: 'Delete team' })[1]);

    await waitFor(() => {
      expect(screen.getByText('Team deleted')).toBeInTheDocument();
    });
  });

  it('shows load error when teams fetch fails', async () => {
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
      if (url === '/api/v1/teams') {
        return jsonResponse({}, 500);
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Could not load teams');
    });
  });
});
