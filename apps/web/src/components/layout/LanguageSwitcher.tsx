import { useTranslation } from 'react-i18next';
import i18n, { persistLocale } from '../../i18n';
import { Button } from '../ui/Button';

export function LanguageSwitcher() {
  const { i18n: i18next } = useTranslation();
  const current = i18next.language.startsWith('ru') ? 'ru' : 'en';

  const setLocale = (locale: 'en' | 'ru') => {
    persistLocale(locale);
    void i18n.changeLanguage(locale);
    document.documentElement.lang = locale;
  };

  return (
    <div className="flex gap-2" role="group" aria-label="Language">
      <Button variant={current === 'en' ? 'primary' : 'ghost'} onClick={() => setLocale('en')}>
        English
      </Button>
      <Button variant={current === 'ru' ? 'primary' : 'ghost'} onClick={() => setLocale('ru')}>
        Русский
      </Button>
    </div>
  );
}
