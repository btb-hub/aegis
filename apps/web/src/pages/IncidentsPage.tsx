import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { IncidentDetail } from '../components/incidents/IncidentDetail';
import { IncidentList } from '../components/incidents/IncidentList';
import type { Incident, IncidentStatus } from '../lib/incidentTypes';

type IncidentsPageProps = {
  incidents: Incident[];
  onAcknowledge: (incidentId: string) => void;
  onResolve: (incidentId: string) => void;
};

export function IncidentsPage({ incidents, onAcknowledge, onResolve }: IncidentsPageProps) {
  const { t } = useTranslation();
  const [statusFilter, setStatusFilter] = useState<IncidentStatus | 'all'>('all');
  const [selectedId, setSelectedId] = useState<string | undefined>(incidents[0]?.id);

  const selectedIncident = useMemo(
    () => incidents.find((item) => item.id === selectedId),
    [incidents, selectedId],
  );

  return (
    <div className="max-w-6xl space-y-6">
      <div>
        <h1 className="text-3xl font-semibold">{t('incidents.page_title')}</h1>
        <p className="text-zinc-600">{t('incidents.page_subtitle')}</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
        <IncidentList
          incidents={incidents}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          onSelect={setSelectedId}
          selectedId={selectedId}
        />
        {selectedIncident ? (
          <IncidentDetail
            incident={selectedIncident}
            onAcknowledge={onAcknowledge}
            onResolve={onResolve}
          />
        ) : (
          <p className="rounded-md border border-dashed border-zinc-200 p-6 text-sm text-zinc-600">
            {t('incidents.select_prompt')}
          </p>
        )}
      </div>
    </div>
  );
}
