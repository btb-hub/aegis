import { describe, expect, it, vi } from 'vitest';
import { persistLocale, resolveLocale } from './i18n';

describe('resolveLocale', () => {
  it('uses stored locale', () => {
    localStorage.setItem('aegis_locale', 'ru');
    expect(resolveLocale()).toBe('ru');
  });

  it('falls back to browser Russian', () => {
    localStorage.removeItem('aegis_locale');
    vi.stubGlobal('navigator', { language: 'ru-RU', languages: ['ru-RU'] });
    expect(resolveLocale()).toBe('ru');
    vi.unstubAllGlobals();
  });

  it('defaults to English', () => {
    localStorage.removeItem('aegis_locale');
    vi.stubGlobal('navigator', { language: 'en-US', languages: ['en-US'] });
    expect(resolveLocale()).toBe('en');
    vi.unstubAllGlobals();
  });
});

describe('persistLocale', () => {
  it('writes localStorage', () => {
    persistLocale('ru');
    expect(localStorage.getItem('aegis_locale')).toBe('ru');
  });
});
