import { useTranslation } from 'react-i18next';
import { Navigate } from 'react-router-dom';
import { LanguageSwitcher } from '../components/layout/LanguageSwitcher';
import { Button } from '../components/ui/Button';
import { useAuth } from '../context/AuthContext';
import { AUTH_PROVIDERS } from '../lib/authTypes';

const providerLabelKey = {
  google: 'auth.sign_in_google',
  slack: 'auth.sign_in_slack',
  express: 'auth.sign_in_express',
} as const;

export function LoginPage() {
  const { t } = useTranslation();
  const { user, loading } = useAuth();

  if (!loading && user) {
    return <Navigate to="/" replace />;
  }

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
          {loading ? (
            <p className="text-center text-sm text-zinc-600">{t('auth.loading')}</p>
          ) : (
            <div className="space-y-3">
              {AUTH_PROVIDERS.map((provider) => (
                <Button
                  key={provider}
                  variant="secondary"
                  href={`/auth/${provider}/login`}
                  className="w-full justify-center"
                >
                  {t(providerLabelKey[provider])}
                </Button>
              ))}
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
