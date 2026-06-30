import { useTranslation } from 'react-i18next';
import type { Incident, IncidentStatus } from '../../lib/incidentTypes';
import { severityLabelKey, severityToTag } from '../../lib/severityTag';
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
  const { t } = useTranslation();

  const filtered =
    statusFilter === 'all' ? incidents : incidents.filter((item) => item.status === statusFilter);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm text-zinc-600">{t('incidents.filter_label')}</span>
        {statusOptions.map((status) => (
          <button
            key={status}
            type="button"
            className={`rounded-md px-3 py-1 text-sm ${
              statusFilter === status ? 'bg-zinc-900 text-white' : 'bg-surface-muted text-zinc-700'
            }`}
            onClick={() => onStatusFilterChange(status)}
          >
            {t(`incidents.status.${status}`)}
          </button>
        ))}
      </div>

      {filtered.length === 0 ? (
        <p className="rounded-md border border-dashed border-zinc-200 p-6 text-sm text-zinc-600">
          {t('incidents.empty_filtered')}
        </p>
      ) : (
        <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 bg-white">
          {filtered.map((incident) => (
            <li key={incident.id}>
              <button
                type="button"
                className={`flex w-full items-start justify-between gap-4 px-4 py-3 text-left hover:bg-zinc-50 ${
                  selectedId === incident.id ? 'bg-surface-muted' : ''
                }`}
                onClick={() => onSelect(incident.id)}
              >
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <SeverityTag
                      severity={severityToTag(incident.severity)}
                      label={t(severityLabelKey(incident.severity), { defaultValue: incident.severity })}
                    />
                    <span className="font-medium">{incident.title}</span>
                  </div>
                  <p className="text-sm text-zinc-600">{incident.id.slice(0, 8)}</p>
                </div>
                <StatusTag
                  variant={incidentStatusVariant(incident.status)}
                  label={t(`incidents.status.${incident.status}`)}
                />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
