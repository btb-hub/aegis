import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import enCommon from './locales/en/common.json';
import ruCommon from './locales/ru/common.json';

const STORAGE_KEY = 'aegis_locale';

export function resolveLocale(): string {
  if (typeof localStorage === 'undefined') {
    return 'en';
  }
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === 'en' || stored === 'ru') {
    return stored;
  }
  const browser = navigator.languages?.[0] ?? navigator.language;
  if (browser?.toLowerCase().startsWith('ru')) {
    return 'ru';
  }
  return 'en';
}

export function persistLocale(locale: string): void {
  if (typeof localStorage === 'undefined') {
    return;
  }
  localStorage.setItem(STORAGE_KEY, locale);
}

void i18n.use(initReactI18next).init({
  resources: {
    en: { common: enCommon },
    ru: { common: ruCommon },
  },
  lng: 'en',
  fallbackLng: 'en',
  defaultNS: 'common',
  interpolation: { escapeValue: false },
});

export default i18n;
