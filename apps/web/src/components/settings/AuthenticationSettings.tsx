import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { settingsRequest, type ProviderDraft, type Settings } from '../../lib/settings';
import { Input } from '../ui/Input';
import { Button } from '../ui/Button';
import { Select } from '../ui/Select';
import { StatusTag } from '../ui/StatusTag';

type Props = { settings?: Settings; email?: string; bootstrap?: boolean; onActivated?: (settings: Settings) => void };

export function AuthenticationSettings({ settings, email = '', bootstrap = false, onActivated }: Props) {
  const { t } = useTranslation();
  const [provider, setProvider] = useState(() => {
    const requested = new URLSearchParams(window.location.search).get('provider');
    return requested && ['google', 'slack', 'express'].includes(requested) ? requested : 'google';
  });
  const [draft, setDraft] = useState<ProviderDraft | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [adminEmail, setAdminEmail] = useState(email);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [dirty, setDirty] = useState(false);
  const base = bootstrap ? '/api/v1/bootstrap' : '/api/v1/settings';
  const prefix = `${provider.toUpperCase()}_OIDC_`;

  const load = useCallback(async () => {
    setDraft(null); setError(null); setMessage(null);
    try {
      const next = await settingsRequest<ProviderDraft>(`${base}/providers/${provider}`);
      setDraft(next); setValues(next.values); setAdminEmail(next.expected_email || email); setDirty(false);
    } catch { setError('settings.request_failed'); }
  }, [base, provider, email]);
  useEffect(() => { void load(); }, [load]);

  async function perform(action: 'save' | 'test' | 'activate' | 'disable') {
    if (!draft) { return; }
    setBusy(true); setError(null); setMessage(null);
    try {
      if (action === 'save') {
        const next = await settingsRequest<ProviderDraft>(`${base}/providers/${provider}`, 'PUT', {
          revision: draft.revision, values, expected_email: bootstrap ? adminEmail : email,
        });
        setDraft(next); setValues(next.values); setDirty(false); setMessage('settings.draft_saved');
      } else if (action === 'test') {
        const result = await settingsRequest<{ authorization_url: string }>(`${base}/providers/${provider}/test`, 'POST');
        window.location.assign(result.authorization_url);
      } else {
        const next = await settingsRequest<Settings>(`${base}/providers/${provider}/activate`, 'POST', {
          revision: settings?.revision, draft_revision: draft.revision, enabled: action === 'activate',
        });
        onActivated?.(next); setMessage('settings.saved');
      }
    } catch (err) { setError(err instanceof Error ? err.message : 'settings.request_failed'); }
    finally { setBusy(false); }
  }

  return <section className="space-y-4 rounded-lg border border-zinc-200 bg-white p-6">
    <h2 className="text-lg font-semibold">{t('settings.authentication')}</h2>
    <p className="text-sm text-zinc-600">{t(bootstrap ? 'settings.first_admin_body' : 'settings.authentication_body')}</p>
    <Select label={t('settings.provider')} value={provider} onChange={setProvider}
      options={['google', 'slack', 'express'].map((id) => ({ value: id, label: t(`account.provider.${id}`) }))} />
    {draft ? <>
      {!bootstrap ? <StatusTag variant={settings?.enabled[provider] ? 'resolved' : 'neutral'} label={t(settings?.enabled[provider] ? 'settings.provider_active' : 'settings.provider_inactive')} /> : null}
      {['CLIENT_ID', 'CLIENT_SECRET', 'REDIRECT_URL', ...(provider === 'express' ? ['ISSUER'] : [])].map((field) => {
        const key = prefix + field;
        return <Input key={key} label={t(`settings.field.${field.toLowerCase()}`)} value={values[key] ?? ''}
          type={field === 'CLIENT_SECRET' ? 'password' : 'text'} autoComplete={field === 'CLIENT_SECRET' ? 'new-password' : undefined}
          hint={draft.secret_present[key] ? t('settings.secret_saved') : undefined}
          onChange={(value) => { setValues((current) => ({ ...current, [key]: value })); setDirty(true); }} />;
      })}
      {bootstrap ? <Input label={t('settings.first_admin_email')} value={adminEmail}
        onChange={(value) => { setAdminEmail(value); setDirty(true); }} /> : null}
      <div className="flex flex-wrap gap-2">
        <Button variant="secondary" disabled={busy || (!dirty && draft.revision > 0)} onClick={() => void perform('save')}>{t('settings.save_draft')}</Button>
        <Button disabled={busy || dirty || draft.revision === 0} onClick={() => void perform('test')}>{t('settings.test_signin')}</Button>
        {!bootstrap ? <>
          <Button variant="secondary" disabled={busy || dirty || !draft.tested} onClick={() => void perform('activate')}>{t('settings.activate')}</Button>
          {settings?.enabled[provider] ? <Button variant="ghost" disabled={busy} onClick={() => void perform('disable')}>{t('settings.disable')}</Button> : null}
        </> : null}
      </div>
      {draft.tested && !dirty ? <p className="text-sm text-zinc-600">{t('settings.tested')}</p> : null}
    </> : error ? <Button variant="secondary" onClick={() => void load()}>{t('settings.retry')}</Button> : <p role="status">{t('settings.loading')}</p>}
    {message ? <p role="status" className="text-sm text-zinc-600">{t(message)}</p> : null}
    {error ? <p role="alert" className="text-sm text-severity-p1">{t(error)}</p> : null}
  </section>;
}
