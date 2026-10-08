import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, Navigate, useSearchParams } from 'react-router-dom';
import { LanguageSwitcher } from '../components/layout/LanguageSwitcher';
import { AppVersion } from '../components/layout/AppVersion';
import { Button } from '../components/ui/Button';
import { useAuth } from '../context/AuthContext';
import { fetchAuthProviders, type AuthProviderId } from '../lib/authTypes';

const providerLabelKey: Record<AuthProviderId, string> = {
  google: 'auth.sign_in_google',
  slack: 'auth.sign_in_slack',
  express: 'auth.sign_in_express',
};

async function fetchDevAuthEnabled(): Promise<boolean> {
  try {
    const response = await fetch('/auth/dev/status');
    if (!response.ok) {
      return false;
    }
    const data = (await response.json()) as { enabled?: boolean };
    return Boolean(data.enabled);
  } catch {
    return false;
  }
}

export function LoginPage() {
  const { t } = useTranslation();
  const { user, loading } = useAuth();
  const [searchParams] = useSearchParams();
  const [devAuthEnabled, setDevAuthEnabled] = useState(false);
  const [providers, setProviders] = useState<AuthProviderId[]>([]);
  const [bootstrapAvailable, setBootstrapAvailable] = useState(false);
  const [devAuthLoading, setDevAuthLoading] = useState(true);
  const devAuthError = searchParams.get('dev_auth_error') === '1';
  const providerUnconfigured = searchParams.get('auth_error') === 'unconfigured';
  const unconfiguredProvider = searchParams.get('provider') ?? '';
  const unconfiguredLabel =
    unconfiguredProvider === 'google' || unconfiguredProvider === 'slack' || unconfiguredProvider === 'express'
      ? t(`account.provider.${unconfiguredProvider}`)
      : unconfiguredProvider;

  useEffect(() => {
    void fetch('/api/v1/bootstrap/status').then(async (response) => {
      if (response.ok) { const status = await response.json() as { available?: boolean }; setBootstrapAvailable(Boolean(status.available)); }
    }).catch(() => undefined);
    void Promise.all([fetchDevAuthEnabled(), fetchAuthProviders()])
      .then(([enabled, configured]) => {
        setDevAuthEnabled(enabled);
        setProviders(configured);
      })
      .finally(() => setDevAuthLoading(false));
  }, []);

  if (!loading && user) {
    return <Navigate to="/" replace />;
  }

  const pageLoading = loading || devAuthLoading;

  return (
    <div className="flex min-h-screen flex-col bg-surface-muted">
      <header className="flex h-14 items-center justify-end border-b border-zinc-200 bg-white px-4">
        <LanguageSwitcher />
      </header>
      <main className="flex flex-1 items-center justify-center p-6">
        <div className="w-full max-w-md space-y-6 rounded-lg border border-zinc-200 bg-white p-8 shadow-sm">
          <div className="space-y-1 text-center">
            <h1 className="text-2xl font-semibold text-zinc-900">{t('app.title')}</h1>
            <p className="text-sm text-zinc-600">{t('auth.login_subtitle')}</p>
          </div>
          {devAuthError ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800" role="alert">
              {t('auth.dev_sign_in_error')}
            </p>
          ) : null}
          {providerUnconfigured ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800" role="alert">
              {t('auth.provider_unconfigured', { provider: unconfiguredLabel })}
            </p>
          ) : null}
          {searchParams.get('settings_error') ? <p role="alert" className="text-sm text-severity-p1">{t('settings.authorization_failed')}</p> : null}
          {bootstrapAvailable ? <Link className="block text-center text-sm text-accent hover:underline" to="/bootstrap">{t('settings.initial_configuration')}</Link> : null}
          {pageLoading ? (
            <p className="text-center text-sm text-zinc-600">{t('auth.loading')}</p>
          ) : (
            <div className="space-y-3">
              {providers.map((provider) => (
                <Button
                  key={provider}
                  variant="secondary"
                  href={`/auth/${provider}/login`}
                  className="w-full justify-center"
                >
                  {t(providerLabelKey[provider])}
                </Button>
              ))}
              {devAuthEnabled ? (
                <div className="space-y-2 border-t border-zinc-200 pt-4">
                  <Button
                    variant="secondary"
                    href="/auth/dev/login?role=admin"
                    className="w-full justify-center"
                  >
                    {t('auth.dev_sign_in')}
                  </Button>
                  <p className="text-center text-xs text-zinc-500">{t('auth.dev_sign_in_hint')}</p>
                </div>
              ) : null}
            </div>
          )}
        </div>
      </main>
      <AppVersion />
    </div>
  );
}
