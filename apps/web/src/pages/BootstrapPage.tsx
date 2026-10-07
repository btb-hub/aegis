import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, Navigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { settingsRequest } from '../lib/settings';
import { AuthenticationSettings } from '../components/settings/AuthenticationSettings';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';

export function BootstrapPage() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const [params] = useSearchParams();
  const [status, setStatus] = useState<{ available: boolean; session_active: boolean } | null>(null);
  const [token, setToken] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => { void settingsRequest<{ available: boolean; session_active: boolean }>('/api/v1/bootstrap/status')
    .then(setStatus).catch(() => setError('settings.request_failed')); }, []);

  async function unlock() {
    setBusy(true); setError(null);
    try {
      await settingsRequest('/api/v1/bootstrap/session', 'POST', { token, public_url: window.location.origin });
      setToken(''); setStatus({ available: true, session_active: true });
    } catch { setToken(''); setError('settings.installation_failed'); }
    finally { setBusy(false); }
  }

  if (user) { return <Navigate to="/settings" replace />; }
  return <main className="mx-auto max-w-2xl space-y-6 p-6">
    <h1 className="text-2xl font-semibold">{t('settings.initial_configuration')}</h1>
    {error || params.get('settings_error') ? <p role="alert" className="text-sm text-severity-p1">{t(error || 'settings.authorization_failed')}</p> : null}
    {!status ? <p role="status">{t('settings.loading')}</p> : !status.available ? <p>{t('settings.installation_closed')}</p> : status.session_active ? <AuthenticationSettings bootstrap /> : <div className="space-y-4 rounded-lg border border-zinc-200 bg-white p-6">
      <p className="text-sm text-zinc-600">{t('settings.installation_body')}</p>
      <Input label={t('settings.installation_token')} type="password" autoComplete="off" value={token} onChange={setToken} />
      <Button disabled={busy || !token} onClick={() => void unlock()}>{t('settings.open_settings')}</Button>
    </div>}
    <Link className="text-sm text-accent hover:underline" to="/login">{t('settings.back_login')}</Link>
  </main>;
}
