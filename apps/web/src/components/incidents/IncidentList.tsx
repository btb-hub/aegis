import { useTranslation } from 'react-i18next';
import type { Incident, IncidentStatus } from '../../lib/incidentTypes';
import { formatDateTime } from '../../lib/formatDate';
import { splitIncidentTitle } from '../../lib/incidentTitle';
import { severityLabelKey, severityToTag } from '../../lib/severityTag';
import { Button } from '../ui/Button';
import { SeverityTag } from '../ui/SeverityTag';
import { StatusTag, incidentStatusVariant } from '../ui/StatusTag';
import { Select } from '../ui/Select';

type IncidentListProps = {
  incidents: Incident[];
  statusFilter: IncidentStatus | 'all';
  onStatusFilterChange: (status: IncidentStatus | 'all') => void;
  onSelect: (incidentId: string) => void;
  selectedId?: string;
};

const statusOptions: Array<IncidentStatus | 'all'> = ['all', 'open', 'acknowledged', 'resolved'];

export function IncidentList({
  incidents,
  statusFilter,
  onStatusFilterChange,
  onSelect,
  selectedId,
}: IncidentListProps) {
  const { t, i18n } = useTranslation();

  const filtered =
    statusFilter === 'all' ? incidents : incidents.filter((item) => item.status === statusFilter);

  return (
    <div className="min-w-0 overflow-hidden rounded-lg border border-zinc-200 bg-white">
      <div className="flex items-center gap-3 border-b border-zinc-200 bg-surface-muted p-3 sm:hidden">
        <label htmlFor="incident-status-filter" className="text-sm font-medium text-zinc-700">{t('incidents.filter_label')}</label>
        <div className="min-w-0 flex-1">
          <Select id="incident-status-filter" label={t('incidents.filter_label')} hideLabel value={statusFilter}
            options={statusOptions.map((status) => ({ value: status, label: t(`incidents.status.${status}`) }))}
            onChange={(status) => onStatusFilterChange(status as IncidentStatus | 'all')} />
        </div>
      </div>
      <div className="hidden flex-wrap gap-1 border-b border-zinc-200 bg-surface-muted p-2 sm:flex" role="group" aria-label={t('incidents.filter_label')}>
        {statusOptions.map((status) => (
          <Button
            key={status}
            variant={statusFilter === status ? 'secondary' : 'ghost'}
            aria-pressed={statusFilter === status}
            className={statusFilter === status ? 'shadow-sm' : '!text-zinc-600'}
            onClick={() => onStatusFilterChange(status)}
          >
            {t(`incidents.status.${status}`)}
          </Button>
        ))}
      </div>

      {filtered.length === 0 ? (
        <div className="space-y-2 px-4 py-8 text-sm text-zinc-600" role="status">
          <p>{t(statusFilter === 'all' ? 'incidents.empty' : 'incidents.empty_filtered')}</p>
          {statusFilter !== 'all' ? <Button variant="ghost" onClick={() => onStatusFilterChange('all')}>{t('incidents.status.all')}</Button> : null}
        </div>
      ) : (
        <ul className="min-w-0 divide-y divide-zinc-200">
          {filtered.map((incident) => {
            const title = splitIncidentTitle(incident.title);

            return (
            <li key={incident.id} className="min-w-0">
              <button
                type="button"
                className={`block w-full min-w-0 border-l-[3px] py-3 pl-3 pr-4 text-left transition-colors hover:bg-zinc-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-inset focus-visible:outline-accent ${
                  selectedId === incident.id ? 'border-accent bg-blue-50/70' : 'border-transparent'
                }`}
                aria-pressed={selectedId === incident.id}
                onClick={() => onSelect(incident.id)}
              >
                <div className="min-w-0 space-y-2">
                  <div className="flex min-w-0 flex-wrap items-center gap-2">
                    <SeverityTag
                      severity={severityToTag(incident.severity)}
                      label={t(severityLabelKey(incident.severity), { defaultValue: incident.severity })}
                    />
                    <StatusTag
                      variant={incidentStatusVariant(incident.status)}
                      label={t(`incidents.status.${incident.status}`)}
                    />
                  </div>
                  <p className="line-clamp-2 break-words text-sm font-medium leading-5 text-zinc-900 [overflow-wrap:anywhere]">
                    {title.summary}
                  </p>
                  <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-zinc-600">
                    <span className="font-mono">{incident.id.slice(0, 8)}</span>
                    <time className="tabular-nums" dateTime={incident.createdAt}>{formatDateTime(new Date(incident.createdAt), i18n.language)}</time>
                  </div>
                </div>
              </button>
            </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
