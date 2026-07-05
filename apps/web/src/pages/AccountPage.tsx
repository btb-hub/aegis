import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Toast } from '../components/ui/Toast';
import { useAuth } from '../context/AuthContext';
import { AUTH_PROVIDERS, createExpressLinkCode, patchAuthMe, type AuthProviderId } from '../lib/authTypes';
import i18n, { persistLocale } from '../i18n';

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return '?';
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return `${parts[0][0]}${parts[1][0]}`.toUpperCase();
}

export function AccountPage() {
  const { t } = useTranslation();
  const { user, refresh } = useAuth();
  const [displayName, setDisplayName] = useState('');
  const [locale, setLocale] = useState<'en' | 'ru'>('en');
  const [savingProfile, setSavingProfile] = useState(false);
  const [savingLocale, setSavingLocale] = useState(false);
  const [profileError, setProfileError] = useState<string | null>(null);
  const [localeError, setLocaleError] = useState<string | null>(null);
  const [expressCommand, setExpressCommand] = useState<string | null>(null);
  const [expressError, setExpressError] = useState<string | null>(null);
  const [generatingCode, setGeneratingCode] = useState(false);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);

  useEffect(() => {
    if (!user) {
      return;
    }
    setDisplayName(user.display_name);
    setLocale(user.locale.startsWith('ru') ? 'ru' : 'en');
  }, [user]);

  const linkedProviders = useMemo(() => {
    const set = new Set((user?.identities ?? []).map((item) => item.provider));
    return set;
  }, [user]);

  const profileDirty = user ? displayName.trim() !== user.display_name : false;

  const saveProfile = useCallback(async () => {
    if (!profileDirty) {
      return;
    }
    setSavingProfile(true);
    setProfileError(null);
    try {
      await patchAuthMe({ display_name: displayName.trim() });
      await refresh();
      setToast({ message: t('account.profile_updated'), variant: 'success' });
    } catch (error) {
      setProfileError(error instanceof Error ? error.message : t('account.profile_save_error'));
    } finally {
      setSavingProfile(false);
    }
  }, [displayName, profileDirty, refresh, t]);

  const saveLocale = useCallback(
    async (next: 'en' | 'ru') => {
      setLocale(next);
      setSavingLocale(true);
      setLocaleError(null);
      try {
        await patchAuthMe({ locale: next });
        persistLocale(next);
        await i18n.changeLanguage(next);
        document.documentElement.lang = next;
        await refresh();
        setToast({ message: t('account.language_updated'), variant: 'success' });
      } catch (error) {
        setLocaleError(error instanceof Error ? error.message : t('account.language_save_error'));
      } finally {
        setSavingLocale(false);
      }
    },
    [refresh, t],
  );

  const generateExpressCode = useCallback(async () => {
    setGeneratingCode(true);
    setExpressError(null);
    try {
      const data = await createExpressLinkCode();
      setExpressCommand(data.command);
    } catch (error) {
      setExpressError(error instanceof Error ? error.message : t('account.express_code_error'));
    } finally {
      setGeneratingCode(false);
    }
  }, [t]);

  if (!user) {
    return (
      <div className="max-w-2xl space-y-4">
        <p className="text-sm text-zinc-600">{t('account.sign_in_required')}</p>
        <Link to="/login?redirect=/account" className="text-sm text-accent hover:underline">
          {t('account.open_login')}
        </Link>
      </div>
    );
  }

  return (
    <PageContent className="mx-auto max-w-2xl">
      <PageHeader
        title={t('account.page_title')}
        subtitle={t('account.page_subtitle')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('account.page_title') }],
        }}
      />

      <section className="space-y-4 rounded-lg border border-zinc-200 bg-white p-6">
        <h2 className="text-lg font-semibold">{t('account.profile_title')}</h2>
        <div className="flex items-center gap-4">
          {user.avatar_url ? (
            <img src={user.avatar_url} alt="" className="h-14 w-14 rounded-full object-cover" />
          ) : (
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-surface text-lg font-semibold text-zinc-700">
              {initials(user.display_name || user.email)}
            </div>
          )}
          <div>
            <p className="text-sm text-zinc-600">{user.email}</p>
            <span className="mt-1 inline-block rounded-full bg-zinc-100 px-2 py-0.5 text-xs font-medium text-zinc-700">
              {t(`account.role.${user.role}`, { defaultValue: user.role })}
            </span>
          </div>
        </div>
        <Input
          label={t('account.display_name_label')}
          value={displayName}
          onChange={setDisplayName}
        />
        {profileError ? <p className="text-sm text-red-700">{profileError}</p> : null}
        <Button disabled={!profileDirty || savingProfile} onClick={() => void saveProfile()}>
          {t('actions.save')}
        </Button>
      </section>

      <section className="space-y-4 rounded-lg border border-zinc-200 bg-white p-6">
        <h2 className="text-lg font-semibold">{t('account.language_title')}</h2>
        <div className="flex gap-2" role="group" aria-label={t('language.switcher_label')}>
          <Button
            variant={locale === 'en' ? 'secondary' : 'ghost'}
            disabled={savingLocale}
            onClick={() => void saveLocale('en')}
          >
            {t('language.en')}
          </Button>
          <Button
            variant={locale === 'ru' ? 'secondary' : 'ghost'}
            disabled={savingLocale}
            onClick={() => void saveLocale('ru')}
          >
            {t('language.ru')}
          </Button>
        </div>
        {localeError ? <p className="text-sm text-red-700">{localeError}</p> : null}
      </section>

      <section className="space-y-4 rounded-lg border border-zinc-200 bg-white p-6">
        <h2 className="text-lg font-semibold">{t('account.connected_title')}</h2>
        <p className="text-sm text-zinc-600">{t('account.connected_body')}</p>
        <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200">
          {AUTH_PROVIDERS.map((provider) => (
            <li key={provider} className="flex items-center justify-between px-4 py-3 text-sm">
              <span className="font-medium capitalize">{provider}</span>
              {linkedProviders.has(provider) ? (
                <span className="text-zinc-600">{t('account.connected')}</span>
              ) : (
                <Link
                  to={`/auth/${provider}/login?redirect=/account`}
                  className="text-accent hover:underline"
                >
                  {t('account.connect_provider', { provider: t(`account.provider.${provider as AuthProviderId}`) })}
                </Link>
              )}
            </li>
          ))}
        </ul>
      </section>

      <section className="space-y-4 rounded-lg border border-zinc-200 bg-white p-6">
        <h2 className="text-lg font-semibold">{t('account.paging_title')}</h2>
        <div className="space-y-2 text-sm">
          <p>
            <span className="font-medium">{t('account.slack_id_label')}:</span>{' '}
            {user.slack_user_id ?? t('account.slack_id_empty')}
          </p>
          <p>
            <span className="font-medium">{t('account.express_id_label')}:</span>{' '}
            {user.express_user_huid ?? t('account.express_id_empty')}
          </p>
        </div>
        {!user.express_user_huid ? (
          <div className="space-y-2">
            <Button variant="secondary" disabled={generatingCode} onClick={() => void generateExpressCode()}>
              {t('account.generate_link_code')}
            </Button>
            {expressCommand ? (
              <p className="rounded-md bg-surface px-3 py-2 font-mono text-sm">{expressCommand}</p>
            ) : null}
            {expressError ? <p className="text-sm text-red-700">{expressError}</p> : null}
          </div>
        ) : null}
      </section>

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
