import { useTranslation } from 'react-i18next';
import type { AlertGroup, AlertItem } from '../../lib/alertTypes';
import { severityLabelKey, severityToTag } from '../../lib/severityTag';
import { DataTable } from '../ui/DataTable';
import { Pagination } from '../ui/Pagination';
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

export function AlertTable({ items, total, page, pageSize, onPageChange }: AlertTableProps) {
  const { t, i18n } = useTranslation();
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="space-y-3">
      <DataTable
        columns={[
          {
            key: 'severity',
            header: t('alerts.column.severity'),
            render: (alert) => (
              <SeverityTag
                severity={severityToTag(alert.severity)}
                label={t(severityLabelKey(alert.severity), { defaultValue: alert.severity })}
              />
            ),
          },
          {
            key: 'title',
            header: t('alerts.column.title'),
            cellClassName: 'font-medium text-zinc-900',
            render: (alert) => alert.title,
          },
          {
            key: 'status',
            header: t('alerts.column.status'),
            render: (alert) => (
              <StatusTag
                variant={alertStatusVariant(alert.status)}
                label={t(`incidents.alert_status.${alert.status}`, { defaultValue: alert.status })}
              />
            ),
          },
          {
            key: 'received_at',
            header: t('alerts.column.received_at'),
            cellClassName: 'text-zinc-600',
            render: (alert) => formatDate(alert.received_at, i18n.language),
          },
        ]}
        rows={items}
        rowKey={(alert) => alert.id}
        emptyMessage={t('alerts.empty')}
      />
      <Pagination
        page={page}
        pageSize={pageSize}
        total={total}
        onPageChange={onPageChange}
        totalLabel={t('alerts.pagination.total', { total })}
        prevLabel={t('alerts.pagination.prev')}
        nextLabel={t('alerts.pagination.next')}
        pageLabel={t('alerts.pagination.page', { page, totalPages })}
      />
    </div>
  );
}

export function AlertGroupTable({ groups, groupBy, total }: AlertGroupTableProps) {
  const { t, i18n } = useTranslation();

  return (
    <div className="space-y-3">
      <p className="text-sm text-zinc-600">{t('alerts.group.summary', { groupBy, total })}</p>
      <DataTable
        columns={[
          {
            key: 'group',
            header: t('alerts.column.group'),
            cellClassName: 'font-medium text-zinc-900',
            render: (group) => group.key || t('alerts.analytics.empty_value'),
          },
          {
            key: 'count',
            header: t('alerts.column.count'),
            cellClassName: 'text-zinc-700',
            render: (group) => group.count,
          },
          {
            key: 'sample',
            header: t('alerts.column.sample'),
            cellClassName: 'text-zinc-700',
            render: (group) =>
              group.sample ? (
                <span>
                  {group.sample.title} · {formatDate(group.sample.received_at, i18n.language)}
                </span>
              ) : (
                '—'
              ),
          },
        ]}
        rows={groups}
        rowKey={(group) => group.key || 'empty'}
        emptyMessage={t('alerts.empty')}
      />
    </div>
  );
}
