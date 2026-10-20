import { useTranslation } from 'react-i18next';
import i18n, { persistLocale } from '../../i18n';
import { patchAuthMe } from '../../lib/authTypes';
import { useAuth } from '../../context/AuthContext';

export function LanguageSwitcher({ appearance = 'light' }: { appearance?: 'light' | 'dark' }) {
  const { t, i18n: i18next } = useTranslation();
  const { user, refresh } = useAuth();
  const current = i18next.language.startsWith('ru') ? 'ru' : 'en';

  const setLocale = (locale: 'en' | 'ru') => {
    persistLocale(locale);
    void i18n.changeLanguage(locale);
    document.documentElement.lang = locale;
    if (user) {
      void patchAuthMe({ locale }).then(() => refresh()).catch(() => undefined);
    }
  };

  return (
    <div className="flex gap-2" role="group" aria-label={t('language.switcher_label')}>
      {(['en', 'ru'] as const).map((locale) => <button
        key={locale}
        type="button"
        aria-pressed={current === locale}
        onClick={() => setLocale(locale)}
        className={`min-h-11 rounded-md border px-3 py-2 text-sm font-medium sm:min-h-9 ${appearance === 'dark'
          ? current === locale ? 'border-white/30 bg-white/10 text-white' : 'border-transparent text-navigation-muted hover:text-white'
          : current === locale ? 'border-zinc-300 bg-white text-zinc-900' : 'border-transparent text-zinc-600 hover:bg-zinc-100'} ${appearance === 'dark' ? 'focus-visible:outline-white' : ''}`}
      >{t(`language.${locale}`)}</button>)}
    </div>
  );
}
