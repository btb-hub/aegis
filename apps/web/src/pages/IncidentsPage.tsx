import { useCallback, useEffect, useMemo, useState } from 'react';
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
  resolveIncident,
  type HandoffTarget,
} from '../lib/incidentsApi';
import type { Incident, IncidentStatus } from '../lib/incidentTypes';
import { fetchTeam } from '../lib/shiftsApi';

export function IncidentsPage() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const initialStatus = (searchParams.get('status') as IncidentStatus | 'all' | null) ?? 'all';

  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selectedId, setSelectedId] = useState<string | undefined>();
  const [selectedIncident, setSelectedIncident] = useState<Incident | null>(null);
  const [handoffTargets, setHandoffTargets] = useState<HandoffTarget[]>([]);
  const [owningTeamName, setOwningTeamName] = useState<string | undefined>();
  const [owningTier, setOwningTier] = useState<string | undefined>();
  const [statusFilter, setStatusFilter] = useState<IncidentStatus | 'all'>(initialStatus);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const loadList = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const items = await fetchIncidents(statusFilter === 'all' ? undefined : statusFilter);
      setIncidents(items);
      if (items.length === 0) {
        setSelectedId(undefined);
        setSelectedIncident(null);
      } else if (!selectedId || !items.some((item) => item.id === selectedId)) {
        setSelectedId(items[0].id);
      }
    } catch {
      setLoadError(t('incidents.load_error'));
      setIncidents([]);
    } finally {
      setLoading(false);
    }
  }, [selectedId, statusFilter, t]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  useEffect(() => {
    if (!selectedId) {
      setSelectedIncident(null);
      setHandoffTargets([]);
      return;
    }
    let cancelled = false;
    const loadDetail = async () => {
      setDetailLoading(true);
      setActionError(null);
      try {
        const detail = await fetchIncidentDetail(selectedId);
        if (cancelled) {
          return;
        }
        const [handoffTargetList, team] = await Promise.all([
          fetchHandoffTargets(detail.teamId),
          fetchTeam(detail.teamId).catch(() => null),
        ]);
        if (cancelled) {
          return;
        }
        setSelectedIncident(detail);
        setHandoffTargets(handoffTargetList);
        setOwningTeamName(team?.name);
        setOwningTier(team?.support_tier);
      } catch {
        if (!cancelled) {
          setActionError(t('incidents.detail_load_error'));
        }
      } finally {
        if (!cancelled) {
          setDetailLoading(false);
        }
      }
    };
    void loadDetail();
    return () => {
      cancelled = true;
    };
  }, [selectedId, t]);

  const refreshDetail = useCallback(async (incidentId: string) => {
    const detail = await fetchIncidentDetail(incidentId);
    setSelectedIncident(detail);
    setHandoffTargets(await fetchHandoffTargets(detail.teamId));
    await loadList();
  }, [loadList]);

  const canBounce = useMemo(
    () =>
      selectedIncident?.timeline.some(
        (event) => event.kind === 'handoff' && !selectedIncident.timeline.some((e) => e.kind === 'bounced'),
      ) ?? false,
    [selectedIncident],
  );

  const runAction = async (action: () => Promise<void>, incidentId: string) => {
    setActionError(null);
    try {
      await action();
      await refreshDetail(incidentId);
    } catch {
      setActionError(t('incidents.action_error'));
    }
  };

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

      {loadError ? <Banner variant="error">{loadError}</Banner> : null}
      {actionError ? <Banner variant="error">{actionError}</Banner> : null}

      {loading ? (
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
          {detailLoading ? (
            <p className="rounded-md border border-dashed border-zinc-200 p-6 text-sm text-zinc-600">
              {t('incidents.loading_detail')}
            </p>
          ) : selectedIncident ? (
            <IncidentDetail
              incident={selectedIncident}
              teams={handoffTargets.map((team) => ({ id: team.id, name: team.name }))}
              owningTeamName={owningTeamName}
              owningTier={owningTier}
              canBounce={canBounce}
              onAcknowledge={(id) => void runAction(() => acknowledgeIncident(id), id)}
              onResolve={(id) => void runAction(() => resolveIncident(id), id)}
              onHandoff={(id, toTeamId, note) => void runAction(() => handoffIncident(id, toTeamId, note), id)}
              onBounce={(id, note) => void runAction(() => bounceIncident(id, note), id)}
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
