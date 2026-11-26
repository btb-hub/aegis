import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '../context/AuthContext';
import i18n from '../i18n';
import { backupErrorKey, BackupsApiError } from '../lib/backupsApi';
import { BackupsPage } from './BackupsPage';

const member = { id: 'u-1', email: 'oncall@example.com', display_name: 'On Call', role: 'member', locale: 'en', provider: 'google' };
const viewer = { ...member, role: 'viewer' };
const admin = { ...member, role: 'admin' };

const passed = {
  backup: 'gamepass', stale: false, review: null,
  check: { id: 'gp-1', backup: 'gamepass', status: 'passed', engine: 'postgres', started_at: '2026-11-25T03:00:00Z', finished_at: '2026-11-25T03:40:00Z', replicas_match: true, restore_ok: true, manifest_counts_match: true },
};
const failed = {
  backup: 'nish', stale: true, review: null,
  check: { id: 'nish-1', backup: 'nish', status: 'failed', engine: 'postgres', started_at: '2026-11-23T03:00:00Z', finished_at: '2026-11-23T03:20:00Z', replicas_match: false, restore_ok: true, manifest_counts_match: false, message: 'orders: 10412 rows, manifest 10420' },
};
const running = {
  backup: 'billing', stale: false, review: null,
  check: { id: 'bill-1', backup: 'billing', status: 'running', engine: '', started_at: '2026-11-25T09:00:00Z', finished_at: null, replicas_match: null, restore_ok: null, manifest_counts_match: null },
};
const reviewed = {
  ...passed, backup: 'ledger', check: { ...passed.check, id: 'led-1', backup: 'ledger' },
  review: { check_id: 'led-1', decision: 'rejected', comment: 'restore was slow', reviewer: { id: 'u-2', display_name: 'Alice', email: 'alice@example.com' }, created_at: '2026-11-25T05:00:00Z' },
};

function json(body: unknown, status = 200): Response {
  return { ok: status >= 200 && status < 300, status, json: async () => body } as Response;
}

type Handler = (url: string, init?: RequestInit) => Response | undefined;

function mockApi(user: unknown, handler: Handler) {
  vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    if (url.includes('/auth/me')) {
      return json(user);
    }
    return handler(url, init) ?? json({}, 404);
  });
}

function renderPage() {
  return render(
    <MemoryRouter>
      <I18nextProvider i18n={i18n}>
        <AuthProvider>
          <BackupsPage />
        </AuthProvider>
      </I18nextProvider>
    </MemoryRouter>,
  );
}

function row(name: string) {
  return screen.getByText(name, { selector: 'span' }).closest('tr') as HTMLElement;
}

describe('BackupsPage', () => {
  beforeEach(async () => {
    vi.stubGlobal('fetch', vi.fn());
    await i18n.changeLanguage('en');
  });
  afterEach(() => vi.unstubAllGlobals());

  it('shows the latest check, its details and the review state', async () => {
    mockApi(member, (url) => (url === '/api/v1/backups' ? json({ stale_after_hours: 26, items: [running, passed, failed, reviewed] }) : undefined));
    renderPage();
    await screen.findByText('gamepass', { selector: 'span' });

    expect(screen.getByText('Checks older than 26 h are marked stale.')).toBeInTheDocument();
    expect(within(row('nish')).getByText('Older than 26 h')).toBeInTheDocument();
    expect(within(row('nish')).getByText('orders: 10412 rows, manifest 10420')).toBeInTheDocument();
    expect(within(row('nish')).getAllByText('Mismatch')).toHaveLength(2);
    expect(within(row('gamepass')).getByText('Waiting for review')).toBeInTheDocument();
    expect(within(row('gamepass')).getByText('postgres')).toBeInTheDocument();
    expect(within(row('billing')).getByText('Checking')).toBeInTheDocument();
    expect(within(row('billing')).getAllByText('Not checked yet')).toHaveLength(3);
    expect(within(row('billing')).getByText('Review after the check finishes')).toBeInTheDocument();
    expect(within(row('billing')).queryByRole('button')).not.toBeInTheDocument();
    expect(within(row('ledger')).getByText('Rejected')).toBeInTheDocument();
    expect(within(row('ledger')).getByText('restore was slow')).toBeInTheDocument();
    expect(within(row('ledger')).getByText(/^Alice · /)).toBeInTheDocument();
    expect(within(row('ledger')).queryByRole('button')).not.toBeInTheDocument();
    expect(within(row('nish')).queryByRole('button', { name: 'Confirm' })).not.toBeInTheDocument();
  });

  it('confirms a passed check', async () => {
    const calls: Array<{ url: string; body: unknown }> = [];
    mockApi(member, (url, init) => {
      if (url === '/api/v1/backups') {
        return json({ stale_after_hours: 26, items: [passed] });
      }
      if (url === '/api/v1/backups/checks/gp-1/confirm') {
        calls.push({ url, body: JSON.parse(String(init?.body)) });
        return json({ check_id: 'gp-1', decision: 'confirmed', comment: '', reviewer: { id: 'u-1', display_name: 'On Call', email: 'oncall@example.com' }, created_at: '2026-11-25T10:00:00Z' }, 201);
      }
      return undefined;
    });
    renderPage();
    fireEvent.click(await screen.findByRole('button', { name: 'Confirm' }));
    await waitFor(() => expect(within(row('gamepass')).getByText('Confirmed', { selector: 'span span' })).toBeInTheDocument());
    expect(calls).toEqual([{ url: '/api/v1/backups/checks/gp-1/confirm', body: {} }]);
    expect(within(row('gamepass')).queryByRole('button', { name: 'Confirm' })).not.toBeInTheDocument();
  });

  it('requires a reason to reject a check', async () => {
    const bodies: unknown[] = [];
    mockApi(member, (url, init) => {
      if (url === '/api/v1/backups') {
        return json({ stale_after_hours: 26, items: [failed] });
      }
      if (url === '/api/v1/backups/checks/nish-1/reject') {
        bodies.push(JSON.parse(String(init?.body)));
        return json({ check_id: 'nish-1', decision: 'rejected', comment: 'counts differ', reviewer: { id: 'u-1', display_name: 'On Call', email: 'oncall@example.com' }, created_at: '2026-11-25T10:00:00Z' }, 201);
      }
      return undefined;
    });
    renderPage();
    fireEvent.click(await screen.findByRole('button', { name: 'Reject' }));
    const dialog = screen.getByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Reject' }));
    expect(within(dialog).getByText('Enter a reason to reject the check.')).toBeInTheDocument();
    expect(bodies).toHaveLength(0);

    fireEvent.change(within(dialog).getByLabelText('Reason'), { target: { value: 'x'.repeat(501) } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Reject' }));
    expect(within(dialog).getByText('Keep the reason to 500 characters or fewer.')).toBeInTheDocument();

    fireEvent.change(within(dialog).getByLabelText('Reason'), { target: { value: '  counts differ ' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Reject' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(bodies).toEqual([{ comment: 'counts differ' }]);
    expect(within(row('nish')).getByText('counts differ')).toBeInTheDocument();
  });

  it('shows the server validation error inside the reject dialog', async () => {
    mockApi(member, (url) => {
      if (url === '/api/v1/backups') {
        return json({ stale_after_hours: 26, items: [failed] });
      }
      return json({ code: 'VALIDATION_ERROR', message: 'comment is required to reject a check' }, 400);
    });
    renderPage();
    fireEvent.click(await screen.findByRole('button', { name: 'Reject' }));
    const dialog = screen.getByRole('dialog');
    fireEvent.change(within(dialog).getByLabelText('Reason'), { target: { value: 'reason' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Reject' }));
    expect(await within(dialog).findByText('Enter a reason to reject the check.')).toBeInTheDocument();
  });

  it('reloads after a conflict', async () => {
    let loads = 0;
    mockApi(member, (url) => {
      if (url === '/api/v1/backups') {
        loads += 1;
        return json({ stale_after_hours: 26, items: [passed] });
      }
      return json({ code: 'CONFLICT', message: 'a newer check exists for this backup; reload the page' }, 409);
    });
    renderPage();
    fireEvent.click(await screen.findByRole('button', { name: 'Confirm' }));
    expect(await screen.findByText('This check changed or already has a review. The page has been refreshed.')).toBeInTheDocument();
    await waitFor(() => expect(loads).toBe(2));
  });

  it('hides review actions from viewers', async () => {
    mockApi(viewer, (url) => (url === '/api/v1/backups' ? json({ stale_after_hours: 26, items: [passed, failed] }) : undefined));
    renderPage();
    await screen.findByText('gamepass', { selector: 'span' });
    expect(screen.queryByRole('button', { name: 'Confirm' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Reject' })).not.toBeInTheDocument();
  });

  it('explains an unreachable checker and lets admins open Settings when not connected', async () => {
    mockApi(admin, () => json({ code: 'BACKUP_CHECKER_NOT_CONFIGURED', message: 'not connected' }, 503));
    renderPage();
    expect(await screen.findByText('The backup checker is not connected. Ask an Aegis administrator to connect it in Settings.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Connect the backup checker in Settings' })).toHaveAttribute('href', '/settings?section=backups');
  });

  it('shows an empty state', async () => {
    mockApi(member, () => json({ stale_after_hours: 26, items: [] }));
    renderPage();
    expect(await screen.findByText('No backup checks yet. Checks appear here after the backup checker runs.')).toBeInTheDocument();
  });

  it('renders Russian copy', async () => {
    await i18n.changeLanguage('ru');
    mockApi({ ...member, locale: 'ru' }, () => json({ stale_after_hours: 26, items: [passed] }));
    renderPage();
    expect(await screen.findByRole('button', { name: 'Подтвердить' })).toBeInTheDocument();
    expect(screen.getByText('Ждёт подтверждения')).toBeInTheDocument();
  });
});

describe('backupErrorKey', () => {
  it('maps API codes to messages', () => {
    const key = (code?: string) => backupErrorKey(new BackupsApiError('x', 500, code), 'fallback');
    expect(key('BACKUP_CHECKER_UNAVAILABLE')).toBe('backups.error.unavailable');
    expect(key('BACKUP_CHECKER_INVALID_RESPONSE')).toBe('backups.error.invalid_response');
    expect(key('FORBIDDEN')).toBe('backups.error.forbidden');
    expect(key('NOT_FOUND')).toBe('backups.error.not_found');
    expect(key('SOMETHING_ELSE')).toBe('fallback');
    expect(backupErrorKey(new Error('network'), 'fallback')).toBe('fallback');
  });
});
