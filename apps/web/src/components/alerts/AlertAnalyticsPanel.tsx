import { useTranslation } from 'react-i18next';
import type { AlertAnalytics } from '../../lib/alertTypes';

type AlertAnalyticsPanelProps = {
  analytics: AlertAnalytics | null;
};

export function AlertAnalyticsPanel({ analytics }: AlertAnalyticsPanelProps) {
  const { t } = useTranslation();
  if (!analytics) {
    return null;
  }

  const severityEntries = Object.entries(analytics.by_severity ?? {});
  const statusEntries = Object.entries(analytics.by_status ?? {});

  return (
    <section
      aria-label={t('alerts.analytics.title')}
      className="rounded-lg border border-zinc-200 bg-white p-4"
    >
      <h2 className="mb-3 text-sm font-semibold text-zinc-900">{t('alerts.analytics.title')}</h2>
      <div className="grid gap-4 md:grid-cols-3">
        <div>
          <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-zinc-500">
            {t('alerts.analytics.by_severity')}
          </h3>
          <ul className="space-y-1 text-sm text-zinc-700">
            {severityEntries.length === 0 ? (
              <li>{t('alerts.analytics.empty')}</li>
            ) : (
              severityEntries.map(([key, count]) => (
                <li key={key} className="flex justify-between gap-4">
                  <span>{t(`incidents.severity.${key}`, { defaultValue: key })}</span>
                  <span className="font-medium">{count}</span>
                </li>
              ))
            )}
          </ul>
        </div>
        <div>
          <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-zinc-500">
            {t('alerts.analytics.by_status')}
          </h3>
          <ul className="space-y-1 text-sm text-zinc-700">
            {statusEntries.map(([key, count]) => (
              <li key={key} className="flex justify-between gap-4">
                <span>{t(`incidents.alert_status.${key}`, { defaultValue: key })}</span>
                <span className="font-medium">{count}</span>
              </li>
            ))}
          </ul>
        </div>
        <div>
          <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-zinc-500">
            {t('alerts.analytics.top_labels')}
          </h3>
          <ul className="space-y-1 text-sm text-zinc-700">
            {(analytics.top_labels ?? []).length === 0 ? (
              <li>{t('alerts.analytics.empty')}</li>
            ) : (
              analytics.top_labels.map((item) => (
                <li key={`${item.key}:${item.value}`} className="flex justify-between gap-4">
                  <span>
                    {item.key}={item.value || t('alerts.analytics.empty_value')}
                  </span>
                  <span className="font-medium">{item.count}</span>
                </li>
              ))
            )}
          </ul>
        </div>
      </div>
    </section>
  );
}
