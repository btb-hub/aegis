import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import {
  authorizePaging, disconnectPaging, fetchPagingConnections, PagingRequestError,
  type PagingConnection, type PagingProvider,
} from '../../lib/pagingConnections';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';
import { Toast } from '../ui/Toast';

const providers: PagingProvider[] = ['slack', 'express'];
const errorCodes = ['setup_required', 'authentication_required', 'bot_required', 'bot_disabled', 'provider_unavailable', 'invalid_authorization', 'cancelled',
  'workspace_mismatch', 'email_unverified', 'email_match_failed', 'identity_in_use'];

export function PagingConnections() {
  const { t } = useTranslation();
  const { user, refresh } = useAuth();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [connections, setConnections] = useState<PagingConnection[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadFailed, setLoadFailed] = useState(false);
  const [busy, setBusy] = useState<PagingProvider | null>(null);
  const [disconnecting, setDisconnecting] = useState<PagingProvider | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);

  const showError = useCallback((code: string) => {
    const known = errorCodes.includes(code) ? code : 'provider_unavailable';
    setError(t(`account.paging_error.${known}`));
  }, [t]);

  const load = useCallback(async () => {
    setLoading(true);
    setLoadFailed(false);
    try {
      setConnections(await fetchPagingConnections());
    } catch {
      setLoadFailed(true);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  useEffect(() => {
    const result = searchParams.get('paging_result');
    const callbackError = searchParams.get('paging_error');
    const provider = searchParams.get('paging_provider') as PagingProvider;
    if (!result && !callbackError) { return; }
    navigate('/account', { replace: true });
    if (callbackError) {
      showError(callbackError);
    } else if (result === 'connected' && providers.includes(provider)) {
      setToast(t('account.paging_connected', { provider: t(`account.provider.${provider}`) }));
      void refresh();
    }
  }, [navigate, refresh, searchParams, showError, t]);

  async function connect(provider: PagingProvider) {
    setBusy(provider);
    setError(null);
    try {
      window.location.assign(await authorizePaging(provider));
    } catch (err) {
      showError(err instanceof PagingRequestError ? err.code : 'provider_unavailable');
      setBusy(null);
    }
  }

  async function disconnect() {
    if (!disconnecting) { return; }
    const provider = disconnecting;
    setBusy(provider);
    setError(null);
    try {
      await disconnectPaging(provider);
      setDisconnecting(null);
      await Promise.all([load(), refresh()]);
      setToast(t('account.paging_disconnected', { provider: t(`account.provider.${provider}`) }));
    } catch (err) {
      setDisconnecting(null);
      showError(err instanceof PagingRequestError ? err.code : 'provider_unavailable');
    } finally {
      setBusy(null);
    }
  }

  return (
    <section className="space-y-4 rounded-lg border border-zinc-200 bg-white p-6">
      <h2 className="text-lg font-semibold">{t('account.paging_title')}</h2>
      <p className="text-sm text-zinc-600">{t('account.paging_body')}</p>
      {loading ? <p role="status" className="text-sm text-zinc-600">{t('account.paging_loading')}</p> : null}
      {loadFailed ? (
        <div className="space-y-2">
          <p role="alert" className="text-sm text-red-700">{t('account.paging_load_error')}</p>
          <Button variant="secondary" onClick={() => void load()}>{t('account.paging_retry')}</Button>
        </div>
      ) : null}
      {!loading && !loadFailed ? (
        <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200">
          {providers.map((provider) => {
            const connection = connections.find((item) => item.provider === provider);
            const identity = connection ? connection.identity : (provider === 'slack' ? user?.slack_user_id : user?.express_user_huid);
            const label = t(`account.provider.${provider}`);
            return (
              <li key={provider} className="space-y-3 px-4 py-4">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="min-w-0">
                    <p className="text-sm font-medium">{label}</p>
                    <p className="text-sm text-zinc-600">{identity ? t('account.connected') : t('account.paging_not_connected')}</p>
                    {identity ? <p className="break-all font-mono text-xs text-zinc-500">{identity}</p> : null}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Button variant="secondary" disabled={busy !== null || !connection?.available} onClick={() => void connect(provider)}>
                      {busy === provider ? t('account.paging_working') : t(identity ? 'account.paging_replace' : 'account.connect_provider', { provider: label })}
                    </Button>
                    {identity ? <Button variant="ghost" disabled={busy !== null} onClick={() => setDisconnecting(provider)}>
                      {t('account.paging_disconnect', { provider: label })}
                    </Button> : null}
                  </div>
                </div>
                {!connection?.available ? <div className="space-y-1">
                  <p className="text-sm text-zinc-600">{t(`account.paging_error.${errorCodes.includes(connection?.unavailable_reason ?? '') ? connection!.unavailable_reason : 'setup_required'}`)}</p>
                  {user?.role === 'admin' ? <Link className="text-sm text-accent hover:underline"
                    to={`/settings?section=${connection?.unavailable_reason === 'authentication_required' ? 'authentication' : 'integrations'}`}>{t('settings.open_settings')}</Link> : null}
                </div> : null}
              </li>
            );
          })}
        </ul>
      ) : null}
      {error ? <p role="alert" className="text-sm text-red-700">{error}</p> : null}
      <Modal open={disconnecting !== null} title={t('account.paging_disconnect', { provider: disconnecting ? t(`account.provider.${disconnecting}`) : '' })}
        onClose={() => { if (!busy) { setDisconnecting(null); } }}
        primaryLabel={t('account.paging_confirm_disconnect')} secondaryLabel={t('account.paging_cancel')}
        primaryDisabled={busy !== null} onPrimary={() => void disconnect()}>
        <p className="text-sm text-zinc-600">{t('account.paging_disconnect_warning', { provider: disconnecting ? t(`account.provider.${disconnecting}`) : '' })}</p>
      </Modal>
      {toast ? <Toast message={toast} variant="success" /> : null}
    </section>
  );
}
