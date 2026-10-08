import { useCallback, useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';
import { settingsRequest, type Settings } from '../lib/settings';
import { AuthenticationSettings } from '../components/settings/AuthenticationSettings';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Modal } from '../components/ui/Modal';
import { IntegrationsPage } from './IntegrationsPage';

const sections = ['authentication', 'integrations', 'behavior', 'deployment', 'diagnostics'];
const fields: Record<string, string[]> = {
  access: ['ADMIN_EMAILS'],
  behavior: ['SESSION_TTL', 'INCIDENT_DEDUP_WINDOW', 'ESCALATION_TIMEOUT', 'ALERT_FINGERPRINT_LABELS'],
  deployment: ['PUBLIC_URL', 'HTTP_ADDR', 'WEBHOOK_SECRET'],
};

export function SettingsPage() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const [params, setParams] = useSearchParams();
  const section = sections.includes(params.get('section') ?? '') ? params.get('section')! : 'authentication';
  const [settings, setSettings] = useState<Settings | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [confirmURL, setConfirmURL] = useState(false);
  const [health, setHealth] = useState<string | null>(null);
  const [alertId, setAlertId] = useState<string | null>(null);
  const isAdmin = user?.role === 'admin';

  const apply = useCallback((next: Settings) => { setSettings(next); setValues(next.values); }, []);
  const load = useCallback(async () => {
    setError(null);
    try { apply(await settingsRequest<Settings>('/api/v1/settings')); }
    catch { setError('settings.request_failed'); }
  }, [apply]);
  useEffect(() => { if (isAdmin) { void load(); } }, [isAdmin, load]);

  async function save(group: string, confirmed = false) {
    if (!settings) { return; }
    if (group === 'deployment' && values.PUBLIC_URL !== settings.values.PUBLIC_URL && !confirmed) { setConfirmURL(true); return; }
    setBusy(true); setError(null); setMessage(null);
    try {
      const patch = Object.fromEntries(fields[group].map((key) => [key, values[key] ?? '']));
      const previousURL = settings.values.PUBLIC_URL;
      const next = await settingsRequest<Settings>(`/api/v1/settings/${group}`, 'PATCH', { revision: settings.revision, values: patch, confirm_public_url: confirmed });
      apply(next); setConfirmURL(false); setMessage(group === 'deployment' && patch.HTTP_ADDR !== settings.values.HTTP_ADDR ? 'settings.saved_restart' : 'settings.saved');
      if (group === 'deployment' && next.values.PUBLIC_URL !== previousURL) { window.location.assign(`${next.values.PUBLIC_URL}/settings?section=deployment`); }
    } catch (err) { setError(err instanceof Error ? err.message : 'settings.request_failed'); setConfirmURL(false); }
    finally { setBusy(false); }
  }

  async function checkHealth() {
    setBusy(true); setError(null);
    try {
      const [live, ready] = await Promise.all([fetch('/healthz'), fetch('/readyz')]);
      setHealth(live.ok && ready.ok ? 'settings.health_ready' : 'settings.health_failed');
    } catch { setHealth('settings.health_failed'); }
    finally { setBusy(false); }
  }

  async function testAlert() {
    setBusy(true); setError(null);
    try { const result = await settingsRequest<{ id: string }>('/api/v1/settings/test-alert', 'POST'); setAlertId(result.id); }
    catch { setError('settings.request_failed'); }
    finally { setBusy(false); }
  }

  const renderFields = (group: string) => <section className="space-y-4 rounded-lg border border-zinc-200 bg-white p-6">
    <h2 className="text-lg font-semibold">{t(`settings.${group}`)}</h2>
    {group === 'deployment' ? <p className="text-sm text-zinc-600">{t('settings.deployment_body')}</p> : null}
    {fields[group].map((key) => <Input key={key} label={t(`settings.field.${key.toLowerCase()}`)}
      value={values[key] ?? ''} type={key === 'WEBHOOK_SECRET' ? 'password' : 'text'}
      hint={settings?.secret_present[key] ? t('settings.secret_saved') : undefined}
      onChange={(value) => { setValues((current) => ({ ...current, [key]: value })); setMessage(null); }} />)}
    <Button disabled={busy} onClick={() => void save(group)}>{t('settings.save')}</Button>
  </section>;

  if (!isAdmin) { return <PageContent><p role="alert">{t('settings.access_denied')}</p></PageContent>; }
  return <PageContent>
    <PageHeader title={t('settings.title')} subtitle={t('settings.body')} />
    <nav aria-label={t('settings.sections')} className="mb-6 flex flex-wrap gap-2">
      {sections.map((id) => <Button key={id} variant={section === id ? 'secondary' : 'ghost'}
        onClick={() => { setParams({ section: id }); setError(null); setMessage(null); }}>{t(`settings.${id}`)}</Button>)}
    </nav>
    {params.get('settings_error') ? <p role="alert" className="mb-4 text-sm text-severity-p1">{t('settings.authorization_failed')}</p> : null}
    {params.get('settings_result') ? <p role="status" className="mb-4 text-sm">{t(params.get('settings_result') === 'installed' ? 'settings.installed' : 'settings.tested')}</p> : null}
    {error ? <div className="mb-4 space-y-2"><p role="alert" className="text-sm text-severity-p1">{t(error)}</p><Button variant="secondary" onClick={() => void load()}>{t('settings.reload')}</Button></div> : null}
    {message ? <p role="status" className="mb-4 text-sm">{t(message)}</p> : null}
    {settings ? <div className="space-y-6">
      {section === 'authentication' ? <><AuthenticationSettings settings={settings} email={user.email} onActivated={apply} />{renderFields('access')}</> : null}
      {section === 'integrations' ? <IntegrationsPage embedded /> : null}
      {section === 'behavior' ? renderFields('behavior') : null}
      {section === 'deployment' ? renderFields('deployment') : null}
      {section === 'diagnostics' ? <section className="space-y-4 rounded-lg border border-zinc-200 bg-white p-6">
        <h2 className="text-lg font-semibold">{t('settings.diagnostics')}</h2>
        <div className="flex flex-wrap gap-2"><Button variant="secondary" disabled={busy} onClick={() => void checkHealth()}>{t('settings.check_health')}</Button>
          <Button disabled={busy} onClick={() => void testAlert()}>{t('settings.test_alert')}</Button></div>
        {health ? <p role="status">{t(health)}</p> : null}
        {alertId ? <Link className="text-accent hover:underline" to={`/alerts?alert_id=${alertId}`}>{t('settings.view_test_alert')}</Link> : null}
        <ul className="space-y-2 text-sm">{['google', 'slack', 'express'].map((provider) => <li key={provider}>{t(`account.provider.${provider}`)} · {t(settings.enabled[provider] ? 'settings.provider_active' : 'settings.provider_inactive')}</li>)}</ul>
        <Link className="block text-sm text-accent hover:underline" to="/settings?section=integrations">{t('settings.check_integrations')}</Link>
        <Link className="block text-sm text-accent hover:underline" to="/account">{t('settings.check_paging')}</Link>
      </section> : null}
    </div> : !error ? <p role="status">{t('settings.loading')}</p> : null}
    <Modal open={confirmURL} title={t('settings.confirm_public_url')} onClose={() => setConfirmURL(false)}
      primaryLabel={t('settings.confirm_save')} secondaryLabel={t('settings.cancel')} primaryDisabled={busy} onPrimary={() => void save('deployment', true)}>
      <p className="mb-3 text-sm">{t('settings.callback_confirmation')}</p>
      <ul className="space-y-2 break-all font-mono text-xs">{['google', 'slack', 'express'].map((provider) => {
        const saved = settings?.values[`${provider.toUpperCase()}_OIDC_REDIRECT_URL`];
        const previousDefault = `${settings?.values.PUBLIC_URL?.replace(/\/+$/, '')}/auth/${provider}/callback`;
        const callback = saved && saved !== previousDefault ? saved : `${values.PUBLIC_URL?.replace(/\/+$/, '')}/auth/${provider}/callback`;
        return <li key={provider}>{callback}</li>;
      })}</ul>
    </Modal>
  </PageContent>;
}
