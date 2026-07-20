import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { UsersPage } from './UsersPage';

const adminUser = {
  id: 'admin-1',
  email: 'admin@example.com',
  display_name: 'Admin',
  role: 'admin',
  locale: 'en',
  provider: 'google',
};

const memberUser = {
  id: 'member-1',
  email: 'member@example.com',
  display_name: 'Member',
  role: 'member',
  locale: 'en',
  provider: 'google',
};

const alice = { id: 'user-alice', email: 'alice@example.com', display_name: 'Alice', role: 'member', role_pinned: false };
const pinnedAdmin = {
  id: 'user-pinned',
  email: 'pinned@example.com',
  display_name: 'Pinned Admin',
  role: 'admin',
  role_pinned: true,
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
    <MemoryRouter>
      <I18nextProvider i18n={i18n}>
        <AuthProvider>
          <UsersPage />
        </AuthProvider>
      </I18nextProvider>
    </MemoryRouter>,
  );
}

function mockFetchAsAdmin(usersBody: unknown, status = 200) {
  vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.includes('/auth/me')) {
      return jsonResponse(adminUser);
    }
    if (url.startsWith('/api/v1/users')) {
      return jsonResponse(usersBody, status);
    }
    return jsonResponse({}, 404);
  });
}

describe('UsersPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('shows a forbidden banner for non-admin users', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return jsonResponse(memberUser);
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Only admins can manage users.');
    });
    expect(fetch).not.toHaveBeenCalledWith(expect.stringContaining('/api/v1/users'), expect.anything());
  });

  it('renders the user list for admins', async () => {
    mockFetchAsAdmin({ items: [alice] });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Alice')).toBeInTheDocument();
    });
    expect(screen.getByText('alice@example.com')).toBeInTheDocument();
  });

  it('changes a role and shows a success toast', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return jsonResponse(adminUser);
      }
      if (url.startsWith('/api/v1/users/user-alice') && init?.method === 'PATCH') {
        return jsonResponse({ ...alice, role: 'admin' });
      }
      if (url.startsWith('/api/v1/users')) {
        return jsonResponse({ items: [alice] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Alice')).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText('Role for Alice'), { target: { value: 'admin' } });

    await waitFor(() => {
      expect(screen.getByText('Role updated')).toBeInTheDocument();
    });
    expect(fetch).toHaveBeenLastCalledWith('/api/v1/users/user-alice', {
      method: 'PATCH',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ role: 'admin' }),
    });
  });

  it('shows the pinned label and disables the select for pinned admins', async () => {
    mockFetchAsAdmin({ items: [pinnedAdmin] });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Pinned Admin')).toBeInTheDocument();
    });
    expect(screen.getByText('Pinned by config')).toBeInTheDocument();
    expect(screen.getByLabelText('Role for Pinned Admin')).toBeDisabled();
  });

  it('shows the API error message when demoting a pinned admin fails', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return jsonResponse(adminUser);
      }
      if (url.startsWith('/api/v1/users/user-pinned') && init?.method === 'PATCH') {
        return jsonResponse(
          {
            code: 'admin_emails_pinned',
            message: 'This user is pinned to admin by ADMIN_EMAILS. Remove the email from ADMIN_EMAILS and restart the API, then demote.',
          },
          409,
        );
      }
      if (url.startsWith('/api/v1/users')) {
        return jsonResponse({ items: [{ ...pinnedAdmin, role_pinned: false }] });
      }
      return jsonResponse({}, 404);
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Pinned Admin')).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText('Role for Pinned Admin'), { target: { value: 'member' } });

    await waitFor(() => {
      expect(
        screen.getByText(
          'This user is pinned to admin by ADMIN_EMAILS. Remove the email from ADMIN_EMAILS and restart the API, then demote.',
        ),
      ).toBeInTheDocument();
    });
  });

  it('shows a load error banner when the list request fails', async () => {
    mockFetchAsAdmin({}, 500);

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Could not load users');
    });
  });
});
