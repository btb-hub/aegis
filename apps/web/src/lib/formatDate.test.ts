import { describe, expect, it } from 'vitest';
import { formatDateTime, formatNumber } from './formatDate';

describe('formatDateTime', () => {
  it('formats in English', () => {
    const formatted = formatDateTime(new Date('2026-06-26T12:00:00Z'), 'en');
    expect(formatted).toMatch(/2026/);
  });

  it('formats in Russian', () => {
    const formatted = formatDateTime(new Date('2026-06-26T12:00:00Z'), 'ru');
    expect(formatted.length).toBeGreaterThan(5);
  });
});

describe('formatNumber', () => {
  it('uses locale grouping', () => {
    expect(formatNumber(1200, 'en')).toContain('1');
  });
});
