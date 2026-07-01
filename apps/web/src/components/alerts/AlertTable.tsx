import { useTranslation } from 'react-i18next';
import type { AlertGroup, AlertItem } from '../../lib/alertTypes';
import { severityLabelKey, severityToTag } from '../../lib/severityTag';
import { SeverityTag } from '../ui/SeverityTag';
import { StatusTag, alertStatusVariant } from '../ui/StatusTag';

type AlertTableProps = {
  items: AlertItem[];
  total: number;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
};

type AlertGroupTableProps = {
  groups: AlertGroup[];
  groupBy: string;
  total: number;
};

function formatDate(value: string, locale: string) {
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}

function AlertRow({ alert, locale }: { alert: AlertItem; locale: string }) {
  const { t, i18n } = useTranslation();
  const lang = locale || i18n.language;

  return (
    <tr>
      <td className="px-4 py-3">
        <SeverityTag
          severity={severityToTag(alert.severity)}
          label={t(severityLabelKey(alert.severity), { defaultValue: alert.severity })}
        />
      </td>
      <td className="px-4 py-3 font-medium text-zinc-900">{alert.title}</td>
      <td className="px-4 py-3">
        <StatusTag
          variant={alertStatusVariant(alert.status)}
          label={t(`incidents.alert_status.${alert.status}`, { defaultValue: alert.status })}
        />
      </td>
      <td className="px-4 py-3 text-zinc-600">{formatDate(alert.received_at, lang)}</td>
    </tr>
  );
}

export function AlertTable({ items, total, page, pageSize, onPageChange }: AlertTableProps) {
  const { t, i18n } = useTranslation();
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="space-y-3">
      <div className="overflow-hidden rounded-lg border border-zinc-200 bg-white">
        <table className="min-w-full divide-y divide-zinc-200 text-sm">
          <thead className="bg-zinc-50 text-left text-zinc-600">
            <tr>
              <th className="px-4 py-3 font-medium">{t('alerts.column.severity')}</th>
              <th className="px-4 py-3 font-medium">{t('alerts.column.title')}</th>
              <th className="px-4 py-3 font-medium">{t('alerts.column.status')}</th>
              <th className="px-4 py-3 font-medium">{t('alerts.column.received_at')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200">
            {items.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-zinc-600">
                  {t('alerts.empty')}
                </td>
              </tr>
            ) : (
              items.map((alert) => <AlertRow key={alert.id} alert={alert} locale={i18n.language} />)
            )}
          </tbody>
        </table>
      </div>
      <div className="flex items-center justify-between text-sm text-zinc-600">
        <span>{t('alerts.pagination.total', { total })}</span>
        <div className="flex items-center gap-2">
          <button
            type="button"
            className="rounded-md px-3 py-1 hover:bg-zinc-100 disabled:opacity-50"
            disabled={page <= 1}
            onClick={() => onPageChange(page - 1)}
          >
            {t('alerts.pagination.prev')}
          </button>
          <span>{t('alerts.pagination.page', { page, totalPages })}</span>
          <button
            type="button"
            className="rounded-md px-3 py-1 hover:bg-zinc-100 disabled:opacity-50"
            disabled={page >= totalPages}
            onClick={() => onPageChange(page + 1)}
          >
            {t('alerts.pagination.next')}
          </button>
        </div>
      </div>
    </div>
  );
}

export function AlertGroupTable({ groups, groupBy, total }: AlertGroupTableProps) {
  const { t, i18n } = useTranslation();

  return (
    <div className="space-y-3">
      <p className="text-sm text-zinc-600">
        {t('alerts.group.summary', { groupBy, total })}
      </p>
      <div className="overflow-hidden rounded-lg border border-zinc-200 bg-white">
        <table className="min-w-full divide-y divide-zinc-200 text-sm">
          <thead className="bg-zinc-50 text-left text-zinc-600">
            <tr>
              <th className="px-4 py-3 font-medium">{t('alerts.column.group')}</th>
              <th className="px-4 py-3 font-medium">{t('alerts.column.count')}</th>
              <th className="px-4 py-3 font-medium">{t('alerts.column.sample')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200">
            {groups.length === 0 ? (
              <tr>
                <td colSpan={3} className="px-4 py-8 text-center text-zinc-600">
                  {t('alerts.empty')}
                </td>
              </tr>
            ) : (
              groups.map((group) => (
                <tr key={group.key}>
                  <td className="px-4 py-3 font-medium text-zinc-900">{group.key || t('alerts.analytics.empty_value')}</td>
                  <td className="px-4 py-3 text-zinc-700">{group.count}</td>
                  <td className="px-4 py-3 text-zinc-700">
                    {group.sample ? (
                      <span>
                        {group.sample.title} ·{' '}
                        {formatDate(group.sample.received_at, i18n.language)}
                      </span>
                    ) : (
                      '—'
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
