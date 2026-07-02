export function formatDateTime(
  value: Date,
  locale: string,
  options?: Intl.DateTimeFormatOptions,
): string {
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    ...options,
  }).format(value);
}

export function formatNumber(value: number, locale: string): string {
  return new Intl.NumberFormat(locale).format(value);
}

export function formatShortDate(value: string | Date, locale = 'en'): string {
  const date = typeof value === 'string' ? new Date(value) : value;
  return new Intl.DateTimeFormat(locale, { month: 'short', day: 'numeric' }).format(date);
}
