import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { IncidentDetail, type HandoffTeamOption } from '../components/incidents/IncidentDetail';
import { IncidentList } from '../components/incidents/IncidentList';
import type { Incident, IncidentStatus } from '../lib/incidentTypes';

type IncidentsPageProps = {
  incidents: Incident[];
  handoffTeams: HandoffTeamOption[];
  onAcknowledge: (incidentId: string) => void;
  onResolve: (incidentId: string) => void;
  onHandoff: (incidentId: string, toTeamId: string, note: string) => void;
  onBounce: (incidentId: string, note: string) => void;
};

export function IncidentsPage({
  incidents,
  handoffTeams,
  onAcknowledge,
  onResolve,
  onHandoff,
  onBounce,
}: IncidentsPageProps) {
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
            teams={handoffTeams}
            canBounce={selectedIncident.timeline.some((event) => event.kind === 'handoff')}
            onAcknowledge={onAcknowledge}
            onResolve={onResolve}
            onHandoff={onHandoff}
            onBounce={onBounce}
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
