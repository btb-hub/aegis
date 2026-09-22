import { useId, useLayoutEffect, useRef, useState } from 'react';
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

function AlertTitle({ title }: { title: string }) {
  const { t } = useTranslation();
  const titleId = useId();
  const titleRef = useRef<HTMLParagraphElement>(null);
  const [expanded, setExpanded] = useState(false);
  const [canExpand, setCanExpand] = useState(title.length > 160);

  useLayoutEffect(() => {
    if (expanded) {
      return undefined;
    }

    const titleElement = titleRef.current;
    if (!titleElement) {
      return undefined;
    }

    const measureOverflow = () => {
      if (titleElement.scrollHeight > 0) {
        setCanExpand(titleElement.scrollHeight > titleElement.clientHeight + 1);
      }
    };

    measureOverflow();

    if (typeof ResizeObserver === 'undefined') {
      return undefined;
    }

    const resizeObserver = new ResizeObserver(measureOverflow);
    resizeObserver.observe(titleElement);
    return () => resizeObserver.disconnect();
  }, [expanded, title]);

  return (
    <div className="min-w-0 max-w-full">
      <p
        ref={titleRef}
        id={titleId}
        className={`${expanded ? '' : 'line-clamp-3'} whitespace-normal break-words leading-5 [overflow-wrap:anywhere]`}
      >
        {title}
      </p>
      {canExpand ? (
        <button
          type="button"
          className="mt-1 rounded-sm text-xs font-medium text-accent hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
          aria-controls={titleId}
          aria-expanded={expanded}
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? t('alerts.title.show_less') : t('alerts.title.show_full')}
        </button>
      ) : null}
    </div>
  );
}

export function AlertTable({ items, total, page, pageSize, onPageChange }: AlertTableProps) {
  const { t, i18n } = useTranslation();
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="space-y-3">
      <DataTable
        tableClassName="table-fixed [width:max(100%,840px)]"
        columns={[
          {
            key: 'severity',
            header: t('alerts.column.severity'),
            headerClassName: 'w-36',
            cellClassName: 'align-top',
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
            cellClassName: 'align-top font-medium text-zinc-900',
            render: (alert) => <AlertTitle title={alert.title} />,
          },
          {
            key: 'status',
            header: t('alerts.column.status'),
            headerClassName: 'w-32',
            cellClassName: 'align-top',
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
            headerClassName: 'w-48',
            cellClassName: 'whitespace-nowrap align-top text-zinc-600',
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
