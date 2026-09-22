import { useTranslation } from 'react-i18next';
import type { Incident, IncidentStatus } from '../../lib/incidentTypes';
import { formatDateTime } from '../../lib/formatDate';
import { splitIncidentTitle } from '../../lib/incidentTitle';
import { severityLabelKey, severityToTag } from '../../lib/severityTag';
import { Button } from '../ui/Button';
import { SeverityTag } from '../ui/SeverityTag';
import { StatusTag, incidentStatusVariant } from '../ui/StatusTag';

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
    <div className="min-w-0 space-y-4">
      <div className="flex flex-wrap items-center gap-2" role="group" aria-label={t('incidents.filter_label')}>
        {statusOptions.map((status) => (
          <Button
            key={status}
            variant={statusFilter === status ? 'secondary' : 'ghost'}
            onClick={() => onStatusFilterChange(status)}
          >
            {t(`incidents.status.${status}`)}
          </Button>
        ))}
      </div>

      {filtered.length === 0 ? (
        <p className="rounded-md border border-dashed border-zinc-200 p-6 text-sm text-zinc-600">
          {t('incidents.empty_filtered')}
        </p>
      ) : (
        <ul className="min-w-0 divide-y divide-zinc-200 overflow-hidden rounded-lg border border-zinc-200 bg-white">
          {filtered.map((incident) => {
            const title = splitIncidentTitle(incident.title);

            return (
            <li key={incident.id} className="min-w-0">
              <button
                type="button"
                className={`block w-full min-w-0 px-4 py-4 text-left transition-colors hover:bg-zinc-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-inset focus-visible:outline-accent ${
                  selectedId === incident.id ? 'bg-blue-50/70' : ''
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
                    <time className="ml-auto text-xs text-zinc-500" dateTime={incident.createdAt}>
                      {formatDateTime(new Date(incident.createdAt), i18n.language)}
                    </time>
                  </div>
                  <p className="line-clamp-3 break-words font-medium leading-5 text-zinc-900 [overflow-wrap:anywhere]">
                    {title.summary}
                  </p>
                  <p className="font-mono text-xs text-zinc-500">{incident.id.slice(0, 8)}</p>
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
