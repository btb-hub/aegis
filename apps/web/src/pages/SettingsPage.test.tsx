import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../i18n';
import { SettingsPage } from './SettingsPage';
import { BootstrapPage } from './BootstrapPage';
import { AuthenticationSettings } from '../components/settings/AuthenticationSettings';
import type { Settings } from '../lib/settings';

let user: { email: string; role: string } | null;
vi.mock('../context/AuthContext', () => ({ useAuth: () => ({ user }) }));

const settings: Settings = {
  values: { PUBLIC_URL: 'http://localhost:3000', HTTP_ADDR: ':8080', WEBHOOK_SECRET: '', ADMIN_EMAILS: 'admin@company.com', SESSION_TTL: '168h', INCIDENT_DEDUP_WINDOW: '24h', ESCALATION_TIMEOUT: '15m', ALERT_FINGERPRINT_LABELS: 'alertname,team' },
  enabled: { google: true }, secret_present: { WEBHOOK_SECRET: true }, revision: 3, bootstrap_closed: true,
};
const draft = { provider: 'google', values: { GOOGLE_OIDC_CLIENT_ID: 'client', GOOGLE_OIDC_CLIENT_SECRET: '', GOOGLE_OIDC_REDIRECT_URL: 'http://localhost:3000/auth/google/callback' }, revision: 1, secret_present: { GOOGLE_OIDC_CLIENT_SECRET: true }, tested: false, expected_email: 'admin@company.com' };
function response(value: unknown, ok = true) { return { ok, json: async () => value } as Response; }
function show(element = <SettingsPage />, path = '/settings') {
  return render(<I18nextProvider i18n={i18n}><MemoryRouter initialEntries={[path]}>{element}</MemoryRouter></I18nextProvider>);
}
function button(key: string) { return screen.getByRole('button', { name: i18n.t(key) }); }

describe('Settings and installation', () => {
  beforeEach(async () => {
    user = { email: 'admin@company.com', role: 'admin' };
    await i18n.changeLanguage('en');
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path.includes('/providers/')) { return response(draft); }
      if (path === '/api/v1/settings') { return response(settings); }
      return response({});
    }));
  });
  afterEach(() => { vi.unstubAllGlobals(); });

  it('keeps secrets blank and requires a saved tested draft before activation', async () => {
    show();
    await screen.findByDisplayValue('client');
    expect(screen.getByLabelText(/^Client secret/)).toHaveValue('');
    expect(button('settings.activate')).toBeDisabled();
    fireEvent.change(screen.getByLabelText('Client ID'), { target: { value: 'replacement' } });
    expect(button('settings.test_signin')).toBeDisabled();
    vi.mocked(fetch).mockResolvedValueOnce(response({ ...draft, revision: 2, values: { ...draft.values, GOOGLE_OIDC_CLIENT_ID: 'replacement' } }));
    fireEvent.click(button('settings.save_draft'));
    await screen.findByText(i18n.t('settings.draft_saved'));
    const call = vi.mocked(fetch).mock.calls.find(([, init]) => init?.method === 'PUT');
    expect(JSON.parse(String(call?.[1]?.body))).toMatchObject({ revision: 1, values: { GOOGLE_OIDC_CLIENT_ID: 'replacement', GOOGLE_OIDC_CLIENT_SECRET: '' } });
    expect(button('settings.test_signin')).toBeEnabled();
  });

  it('activates a verified draft using both revisions and disables providers through the same API', async () => {
    vi.mocked(fetch).mockImplementation(async (input, init) => response(String(input).includes('/providers/') && init?.method !== 'POST' ? { ...draft, tested: true } : settings));
    show(); await screen.findByText(i18n.t('settings.tested'));
    fireEvent.click(button('settings.activate'));
    await screen.findByText(i18n.t('settings.saved'));
    expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/v1/settings/providers/google/activate', expect.objectContaining({ body: JSON.stringify({ revision: 3, draft_revision: 1, enabled: true }) }));
    fireEvent.click(button('settings.disable'));
    await waitFor(() => expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/v1/settings/providers/google/activate', expect.objectContaining({ body: JSON.stringify({ revision: 3, draft_revision: 1, enabled: false }) })));
  });

  it('shows a conflict without losing the current form', async () => {
    show(); await screen.findByDisplayValue('client');
    fireEvent.change(screen.getByLabelText('Client ID'), { target: { value: 'replacement' } });
    vi.mocked(fetch).mockResolvedValueOnce(response({ code: 'CONFLICT' }, false));
    fireEvent.click(button('settings.save_draft'));
    await screen.findByText(i18n.t('settings.conflict'));
    expect(screen.getByLabelText('Client ID')).toHaveValue('replacement');
  });

  it('loads eXpress fields when the provider changes', async () => {
    show(); await screen.findByDisplayValue('client');
    fireEvent.change(screen.getByLabelText('Sign-in provider'), { target: { value: 'express' } });
    await screen.findByLabelText('Issuer URL');
    expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/v1/settings/providers/express', expect.anything());
  });

  it('lets the administrator retry a failed draft load', async () => {
    vi.mocked(fetch).mockRejectedValueOnce(new Error('offline'));
    show(<AuthenticationSettings settings={settings} />);
    await screen.findByRole('alert');
    fireEvent.click(button('settings.retry'));
    await screen.findByDisplayValue('client');
  });

  it('reports a failed sign-in start', async () => {
    show(); await screen.findByDisplayValue('client');
    vi.mocked(fetch).mockRejectedValueOnce(new Error('upstream private response'));
    fireEvent.click(button('settings.test_signin'));
    await screen.findByText(i18n.t('settings.request_failed'));
    expect(screen.queryByText('upstream private response')).not.toBeInTheDocument();
  });

  it('saves behavior for subsequent operations using a revision', async () => {
    show(<SettingsPage />, '/settings?section=behavior');
    await screen.findByDisplayValue('168h');
    fireEvent.change(screen.getByLabelText(i18n.t('settings.field.session_ttl')), { target: { value: '24h' } });
    vi.mocked(fetch).mockResolvedValueOnce(response({ ...settings, revision: 4, values: { ...settings.values, SESSION_TTL: '24h' } }));
    fireEvent.click(button('settings.save'));
    await screen.findByText(i18n.t('settings.saved'));
    expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/v1/settings/behavior', expect.objectContaining({ method: 'PATCH', body: expect.stringContaining('"revision":3') }));
  });

  it('shows an API restart notice for a changed listening address', async () => {
    show(<SettingsPage />, '/settings?section=deployment');
    await screen.findByDisplayValue(':8080');
    expect(screen.getByLabelText(/^Alert webhook secret/)).toHaveValue('');
    fireEvent.change(screen.getByLabelText('Server listening address'), { target: { value: ':8081' } });
    vi.mocked(fetch).mockResolvedValueOnce(response({ ...settings, values: { ...settings.values, HTTP_ADDR: ':8081' } }));
    fireEvent.click(button('settings.save'));
    await screen.findByText(i18n.t('settings.saved_restart'));
  });

  it('requires public URL confirmation and displays callbacks before making any change', async () => {
    show(<SettingsPage />, '/settings?section=deployment');
    await screen.findByDisplayValue(':8080');
    fireEvent.change(screen.getByLabelText('Public application URL'), { target: { value: 'https://new.example' } });
    fireEvent.click(button('settings.save'));
    await screen.findByRole('dialog');
    expect(screen.getByText('https://new.example/auth/express/callback')).toBeInTheDocument();
    expect(vi.mocked(fetch).mock.calls.some(([, init]) => init?.method === 'PATCH')).toBe(false);
    fireEvent.click(button('settings.cancel'));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    fireEvent.click(button('settings.save'));
    vi.mocked(fetch).mockResolvedValueOnce(response({ code: 'CONFLICT' }, false));
    fireEvent.click(button('settings.confirm_save'));
    await screen.findByText(i18n.t('settings.conflict'));
    expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/v1/settings/deployment', expect.objectContaining({ body: expect.stringContaining('"confirm_public_url":true') }));
  });

  it('runs health checks and sends a test alert from Diagnostics', async () => {
    show(<SettingsPage />, '/settings?section=diagnostics');
    await screen.findByRole('button', { name: i18n.t('settings.check_health') });
    fireEvent.click(button('settings.check_health'));
    await screen.findByText(i18n.t('settings.health_ready'));
    vi.mocked(fetch).mockResolvedValueOnce(response({ id: 'test-alert' }));
    fireEvent.click(button('settings.test_alert'));
    expect(await screen.findByRole('link', { name: i18n.t('settings.view_test_alert') })).toHaveAttribute('href', '/alerts?alert_id=test-alert');
    expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/v1/settings/test-alert', expect.objectContaining({ method: 'POST' }));
  });

  it('reports failed diagnostics and allows settings to be reloaded', async () => {
    show(<SettingsPage />, '/settings?section=diagnostics');
    await screen.findByRole('button', { name: i18n.t('settings.check_health') });
    vi.mocked(fetch).mockRejectedValueOnce(new Error('offline'));
    fireEvent.click(button('settings.check_health')); await screen.findByText(i18n.t('settings.health_failed'));
    vi.mocked(fetch).mockRejectedValueOnce(new Error('offline'));
    fireEvent.click(button('settings.test_alert')); await screen.findByText(i18n.t('settings.request_failed'));
    fireEvent.click(button('settings.reload')); await screen.findByRole('button', { name: i18n.t('settings.check_health') });
  });

  it('does not load admin settings for a member', () => {
    user = { email: 'member@company.com', role: 'member' }; show();
    expect(screen.getByRole('alert')).toHaveTextContent(i18n.t('settings.access_denied'));
    expect(fetch).not.toHaveBeenCalled();
  });

  it('renders Russian copy and saves database admin email rules', async () => {
    await i18n.changeLanguage('ru'); show();
    await screen.findByDisplayValue('client');
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Настройки');
    fireEvent.change(screen.getByLabelText(i18n.t('settings.field.admin_emails')), { target: { value: 'second@company.com' } });
    vi.mocked(fetch).mockResolvedValueOnce(response(settings)); fireEvent.click(button('settings.save'));
    await screen.findByText(i18n.t('settings.saved'));
    expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/v1/settings/access', expect.objectContaining({ body: expect.stringContaining('second@company.com') }));
    fireEvent.click(button('settings.behavior')); await screen.findByDisplayValue('168h');
  });

  it('keeps bootstrap tokens out of URLs and clears the field after opening access', async () => {
    user = null;
    vi.mocked(fetch).mockImplementation(async (input) => response(String(input).includes('/providers/') ? draft : { available: true, session_active: false }));
    show(<BootstrapPage />, '/bootstrap');
    await screen.findByLabelText('Installation token');
    fireEvent.change(screen.getByLabelText('Installation token'), { target: { value: 'private-installation-token' } });
    fireEvent.click(button('settings.open_settings'));
    await screen.findByLabelText('First administrator email');
    expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/v1/bootstrap/session', expect.objectContaining({ method: 'POST', body: expect.stringContaining('private-installation-token') }));
    expect(screen.queryByDisplayValue('private-installation-token')).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('First administrator email'), { target: { value: 'first@company.com' } });
    vi.mocked(fetch).mockResolvedValueOnce(response(draft)); fireEvent.click(button('settings.save_draft'));
    await screen.findByText(i18n.t('settings.draft_saved'));
    expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/v1/bootstrap/providers/google', expect.objectContaining({ body: expect.stringContaining('first@company.com') }));
    expect(screen.queryByRole('button', { name: i18n.t('settings.activate') })).not.toBeInTheDocument();
  });

  it('reports rejected bootstrap tokens and cannot reopen a completed installation', async () => {
    user = null; vi.mocked(fetch).mockResolvedValueOnce(response({ available: true, session_active: false }));
    const view = show(<BootstrapPage />); await screen.findByLabelText('Installation token');
    fireEvent.change(screen.getByLabelText('Installation token'), { target: { value: 'wrong' } });
    vi.mocked(fetch).mockResolvedValueOnce(response({}, false)); fireEvent.click(button('settings.open_settings'));
    await screen.findByText(i18n.t('settings.installation_failed')); expect(screen.getByLabelText('Installation token')).toHaveValue('');
    view.unmount(); vi.mocked(fetch).mockResolvedValueOnce(response({ available: false, session_active: false }));
    show(<BootstrapPage />); await screen.findByText(i18n.t('settings.installation_closed'));
    expect(screen.queryByLabelText('Installation token')).not.toBeInTheDocument();
  });
});
