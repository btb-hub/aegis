import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { WorkspacesPage } from './WorkspacesPage';

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
          <WorkspacesPage />
        </AuthProvider>
      </I18nextProvider>
    </MemoryRouter>,
  );
}

describe('WorkspacesPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('lists workspaces with counts for admin', async () => {
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
      if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: '00000000-0000-0000-0000-000000000001',
              name: 'Default',
              slug: 'default',
              description: '',
              team_count: 2,
              routing_rule_count: 1,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Default')).toBeInTheDocument();
    });
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create workspace' })).toBeInTheDocument();
  });

  it('creates a workspace', async () => {
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
      if (url === '/api/v1/workspaces' && init?.method === 'POST') {
        return jsonResponse(
          {
            id: 'ws-new',
            name: 'Platform',
            slug: 'platform',
            description: 'Core',
            created_at: '',
            updated_at: '',
          },
          201,
        );
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({ items: [] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('No workspaces yet')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Create workspace' }));
    fireEvent.change(screen.getByLabelText('Workspace name'), { target: { value: 'Platform' } });
    fireEvent.change(screen.getByLabelText('Description'), { target: { value: 'Core' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save workspace' }));

    await waitFor(() => {
      expect(screen.getByText('Workspace created')).toBeInTheDocument();
    });
  });

  it('deletes an empty workspace as admin', async () => {
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
      if (url === '/api/v1/workspaces/ws-del' && init?.method === 'DELETE') {
        return jsonResponse({}, 204);
      }
      if (url === '/api/v1/workspaces') {
        return jsonResponse({
          items: [
            {
              id: 'ws-del',
              name: 'Temp',
              slug: 'temp',
              description: '',
              team_count: 0,
              routing_rule_count: 0,
              created_at: '',
              updated_at: '',
            },
          ],
        });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Temp')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Delete workspace' }));
    fireEvent.click(screen.getAllByRole('button', { name: 'Delete workspace' })[1]);

    await waitFor(() => {
      expect(screen.getByText('Workspace deleted')).toBeInTheDocument();
    });
  });
});
