import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { IncidentDetail } from '../components/incidents/IncidentDetail';
import { IncidentList } from '../components/incidents/IncidentList';
import { Banner } from '../components/ui/Banner';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import {
  acknowledgeIncident,
  bounceIncident,
  fetchHandoffTargets,
  fetchIncidentDetail,
  fetchIncidents,
  handoffIncident,
  IncidentApiError,
  resolveIncident,
  type HandoffTarget,
} from '../lib/incidentsApi';
import type { Incident, IncidentStatus } from '../lib/incidentTypes';
import { fetchTeam } from '../lib/shiftsApi';
import { resolveApiErrorMessage } from '../lib/apiErrors';
import { incidentQueryKeys } from '../lib/queryClient';
import { validEscalationTargetTiers, type SupportTier } from '../lib/teamTypes';

type IncidentContext = {
  incident: Incident;
  handoffTargets: HandoffTarget[];
  owningTeamName?: string;
  owningTier?: string;
};

async function loadIncidentContext(id: string): Promise<IncidentContext> {
  const detail = await fetchIncidentDetail(id);
  const [handoffTargetList, team] = await Promise.all([
    fetchHandoffTargets(detail.teamId),
    fetchTeam(detail.teamId).catch(() => null),
  ]);
  const allowedTiers = team?.support_tier
    ? validEscalationTargetTiers(team.support_tier as SupportTier)
    : [];
  const handoffTargets =
    allowedTiers.length === 0
      ? handoffTargetList
      : handoffTargetList.filter(
          (target) =>
            target.support_tier && allowedTiers.includes(target.support_tier as SupportTier),
        );
  return {
    incident: detail,
    handoffTargets,
    owningTeamName: team?.name,
    owningTier: team?.support_tier,
  };
}

export function IncidentsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const initialStatus = (searchParams.get('status') as IncidentStatus | 'all' | null) ?? 'all';

  const [selectedId, setSelectedId] = useState<string | undefined>();
  const [statusFilter, setStatusFilter] = useState<IncidentStatus | 'all'>(initialStatus);
  const [actionError, setActionError] = useState<string | null>(null);

  const listQuery = useQuery({
    queryKey: incidentQueryKeys.list(statusFilter),
    queryFn: () => fetchIncidents(statusFilter === 'all' ? undefined : statusFilter),
  });

  const incidents = listQuery.data ?? [];

  useEffect(() => {
    if (!listQuery.data) {
      return;
    }
    const items = listQuery.data;
    if (items.length === 0) {
      setSelectedId(undefined);
      return;
    }
    if (!selectedId || !items.some((item) => item.id === selectedId)) {
      setSelectedId(items[0].id);
    }
  }, [listQuery.data, selectedId]);

  const detailQuery = useQuery({
    queryKey: incidentQueryKeys.detail(selectedId ?? ''),
    queryFn: () => loadIncidentContext(selectedId!),
    enabled: Boolean(selectedId),
  });

  const selectedIncident = detailQuery.data?.incident ?? null;
  const handoffTargets = detailQuery.data?.handoffTargets ?? [];
  const owningTeamName = detailQuery.data?.owningTeamName;
  const owningTier = detailQuery.data?.owningTier;

  const invalidateIncidents = () =>
    queryClient.invalidateQueries({ queryKey: incidentQueryKeys.all });

  const onActionError = (error: Error) => {
    setActionError(
      error instanceof IncidentApiError
        ? resolveApiErrorMessage(t, error.body, t('incidents.action_error'))
        : t('incidents.action_error'),
    );
  };

  const acknowledgeMut = useMutation({
    mutationFn: acknowledgeIncident,
    onSuccess: () => {
      setActionError(null);
      return invalidateIncidents();
    },
    onError: onActionError,
  });
  const resolveMut = useMutation({
    mutationFn: resolveIncident,
    onSuccess: () => {
      setActionError(null);
      return invalidateIncidents();
    },
    onError: onActionError,
  });
  const handoffMut = useMutation({
    mutationFn: ({ id, toTeamId, note }: { id: string; toTeamId: string; note: string }) =>
      handoffIncident(id, toTeamId, note),
    onSuccess: () => {
      setActionError(null);
      return invalidateIncidents();
    },
    onError: onActionError,
  });
  const bounceMut = useMutation({
    mutationFn: ({ id, note }: { id: string; note: string }) => bounceIncident(id, note),
    onSuccess: () => {
      setActionError(null);
      return invalidateIncidents();
    },
    onError: onActionError,
  });

  const canBounce = useMemo(
    () =>
      selectedIncident?.timeline.some(
        (event) => event.kind === 'handoff' && !selectedIncident.timeline.some((e) => e.kind === 'bounced'),
      ) ?? false,
    [selectedIncident],
  );

  return (
    <PageContent>
      <PageHeader
        title={t('incidents.page_title')}
        subtitle={t('incidents.page_subtitle')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('nav.incidents') }],
        }}
      />

      {listQuery.isError ? <Banner variant="error">{t('incidents.load_error')}</Banner> : null}
      {detailQuery.isError ? <Banner variant="error">{t('incidents.detail_load_error')}</Banner> : null}
      {actionError ? <Banner variant="error">{actionError}</Banner> : null}

      {listQuery.isLoading ? (
        <p className="text-sm text-zinc-600">{t('incidents.loading')}</p>
      ) : (
        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
          <IncidentList
            incidents={incidents}
            statusFilter={statusFilter}
            onStatusFilterChange={setStatusFilter}
            onSelect={setSelectedId}
            selectedId={selectedId}
          />
          {selectedId && detailQuery.isLoading ? (
            <p className="rounded-md border border-dashed border-zinc-200 p-6 text-sm text-zinc-600">
              {t('incidents.loading_detail')}
            </p>
          ) : selectedIncident ? (
            <IncidentDetail
              incident={selectedIncident}
              teams={handoffTargets.map((team) => ({
                id: team.id,
                name: team.name,
                supportTier: team.support_tier,
              }))}
              owningTeamName={owningTeamName}
              owningTier={owningTier}
              canBounce={canBounce}
              onAcknowledge={(id) => {
                setActionError(null);
                acknowledgeMut.mutate(id);
              }}
              onResolve={(id) => {
                setActionError(null);
                resolveMut.mutate(id);
              }}
              onHandoff={(id, toTeamId, note) => {
                setActionError(null);
                handoffMut.mutate({ id, toTeamId, note });
              }}
              onBounce={(id, note) => {
                setActionError(null);
                bounceMut.mutate({ id, note });
              }}
            />
          ) : (
            <p className="rounded-md border border-dashed border-zinc-200 p-6 text-sm text-zinc-600">
              {t('incidents.select_prompt')}
            </p>
          )}
        </div>
      )}
    </PageContent>
  );
}
