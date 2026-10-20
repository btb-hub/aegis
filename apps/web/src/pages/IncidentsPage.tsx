import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { IncidentDetail } from '../components/incidents/IncidentDetail';
import { IncidentList } from '../components/incidents/IncidentList';
import { Banner } from '../components/ui/Banner';
import { Button } from '../components/ui/Button';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import {
  acknowledgeIncident, bounceIncident, fetchHandoffTargets, fetchIncidentDetail,
  fetchIncidents, handoffIncident, IncidentApiError, resolveIncident, type HandoffTarget,
} from '../lib/incidentsApi';
import type { Incident, IncidentStatus } from '../lib/incidentTypes';
import { fetchTeam } from '../lib/shiftsApi';
import { resolveApiErrorMessage } from '../lib/apiErrors';
import { validEscalationTargetTiers, type SupportTier } from '../lib/teamTypes';

function parseStatus(value: string | null): IncidentStatus | 'all' {
  return value === 'open' || value === 'acknowledged' || value === 'resolved' ? value : 'all';
}

async function incidentContext(incident: Incident) {
  const [targets, team] = await Promise.all([
    fetchHandoffTargets(incident.teamId), fetchTeam(incident.teamId).catch(() => null),
  ]);
  const allowedTiers = team?.support_tier
    ? validEscalationTargetTiers(team.support_tier as SupportTier) : [];
  return {
    targets: allowedTiers.length === 0 ? targets : targets.filter(
      (target) => target.support_tier && allowedTiers.includes(target.support_tier as SupportTier),
    ),
    team,
  };
}

export function IncidentsPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const linkedIncidentId = searchParams.get('incident_id') || undefined;
  const statusFilter = parseStatus(searchParams.get('status'));
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [lastSelectedId, setLastSelectedId] = useState(linkedIncidentId);
  const [retainedIncidentId, setRetainedIncidentId] = useState<string | undefined>();
  const selectedId = linkedIncidentId
    ?? retainedIncidentId
    ?? incidents.find((incident) => incident.id === lastSelectedId)?.id
    ?? incidents[0]?.id;
  const [selectedIncident, setSelectedIncident] = useState<Incident | null>(null);
  const [handoffTargets, setHandoffTargets] = useState<HandoffTarget[]>([]);
  const [owningTeamName, setOwningTeamName] = useState<string | undefined>();
  const [owningTier, setOwningTier] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [hasLoaded, setHasLoaded] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailRetry, setDetailRetry] = useState(0);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const listRequest = useRef(0);
  const selectedIdRef = useRef(selectedId);
  selectedIdRef.current = selectedId;
  const queueRef = useRef<HTMLDivElement>(null);
  const detailRef = useRef<HTMLElement>(null);
  const returnFocusRef = useRef<HTMLElement | null>(null);
  const previousLinkedId = useRef<string | undefined>();
  const navigation = useRef({ key: location.key, version: 0 });
  if (navigation.current.key !== location.key) {
    navigation.current = { key: location.key, version: navigation.current.version + 1 };
  }

  const loadList = useCallback(async () => {
    const request = ++listRequest.current;
    setLoading(true);
    setLoadError(null);
    try {
      const items = await fetchIncidents(statusFilter === 'all' ? undefined : statusFilter);
      if (request !== listRequest.current) { return; }
      setIncidents(items);
      setHasLoaded(true);
    } catch {
      if (request === listRequest.current) { setLoadError(t('incidents.load_error')); }
    } finally {
      if (request === listRequest.current) { setLoading(false); }
    }
  }, [statusFilter, t]);
  const loadListRef = useRef(loadList);
  loadListRef.current = loadList;

  useEffect(() => {
    void loadList();
    return () => { listRequest.current += 1; };
  }, [loadList]);

  useEffect(() => {
    setRetainedIncidentId(undefined);
  }, [location.key]);

  useEffect(() => {
    if (linkedIncidentId) {
      setLastSelectedId(linkedIncidentId);
      detailRef.current?.focus();
    } else if (previousLinkedId.current) {
      const target = returnFocusRef.current?.isConnected ? returnFocusRef.current
        : queueRef.current?.querySelector<HTMLElement>('button[aria-pressed="true"], button');
      target?.focus();
    }
    previousLinkedId.current = linkedIncidentId;
  }, [linkedIncidentId]);

  useEffect(() => {
    setSelectedIncident(null);
    setHandoffTargets([]);
    setOwningTeamName(undefined);
    setOwningTier(undefined);
    setDetailError(null);
    setActionError(null);
    if (!selectedId) {
      setDetailLoading(false);
      return;
    }
    let cancelled = false;
    const loadDetail = async () => {
      setDetailLoading(true);
      try {
        const detail = await fetchIncidentDetail(selectedId);
        if (cancelled) { return; }
        const context = await incidentContext(detail);
        if (cancelled) { return; }
        setSelectedIncident(detail);
        setHandoffTargets(context.targets);
        setOwningTeamName(context.team?.name);
        setOwningTier(context.team?.support_tier);
      } catch {
        if (!cancelled) { setDetailError(t('incidents.detail_load_error')); }
      } finally {
        if (!cancelled) { setDetailLoading(false); }
      }
    };
    void loadDetail();
    return () => { cancelled = true; };
  }, [selectedId, detailRetry, t]);

  const visibleIncident = selectedIncident?.id === selectedId ? selectedIncident : null;
  const canBounce = useMemo(
    () => visibleIncident?.timeline.some(
      (event) => event.kind === 'handoff' && !visibleIncident.timeline.some((e) => e.kind === 'bounced'),
    ) ?? false,
    [visibleIncident],
  );

  const selectIncident = (incidentId: string) => {
    if (document.activeElement instanceof HTMLElement && queueRef.current?.contains(document.activeElement)) {
      returnFocusRef.current = document.activeElement;
    }
    setSearchParams((previous) => {
      const next = new URLSearchParams(previous);
      next.set('incident_id', incidentId);
      return next;
    });
  };

  const changeStatus = (status: IncidentStatus | 'all') => {
    setSearchParams((previous) => {
      const next = new URLSearchParams(previous);
      next.delete('incident_id');
      if (status === 'all') { next.delete('status'); } else { next.set('status', status); }
      return next;
    });
  };

  const runAction = async (action: () => Promise<void>, incidentId: string): Promise<boolean> => {
    const startedNavigationVersion = navigation.current.version;
    setActionError(null);
    try {
      await action();
    } catch (error) {
      if (selectedIdRef.current === incidentId) {
        setActionError(error instanceof IncidentApiError
          ? resolveApiErrorMessage(t, error.body, t('incidents.action_error'))
          : t('incidents.action_error'));
      }
      return false;
    }
    try {
      const detail = await fetchIncidentDetail(incidentId);
      const context = await incidentContext(detail);
      if (selectedIdRef.current === incidentId) {
        if (navigation.current.version === startedNavigationVersion) {
          // Keep the same detail mounted if the refreshed filter excludes this incident
          // before the router commits its explicit selection.
          setRetainedIncidentId(incidentId);
          setSearchParams((previous) => {
            const next = new URLSearchParams(previous);
            next.set('incident_id', incidentId);
            return next;
          }, { replace: true });
        }
        setSelectedIncident(detail);
        setHandoffTargets(context.targets);
        setOwningTeamName(context.team?.name);
        setOwningTier(context.team?.support_tier);
      }
      await loadListRef.current();
    } catch {
      if (selectedIdRef.current === incidentId) { setActionError(t('incidents.refresh_error')); }
    }
    return true;
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
      <div className="grid min-w-0 items-start gap-6 xl:grid-cols-[minmax(20rem,0.8fr)_minmax(0,1.35fr)]">
        <div
          ref={queueRef}
          data-incident-queue
          className={`min-w-0 space-y-3 ${linkedIncidentId ? 'hidden xl:block' : ''}`}
          aria-busy={loading}
        >
          {loadError ? <div className="space-y-2">
            <Banner variant="error">{loadError}</Banner>
            <Button variant="secondary" onClick={() => void loadList()}>{t('incidents.retry')}</Button>
          </div> : null}
          {loading ? <p role="status" className="text-sm text-zinc-600">
            {t(hasLoaded ? 'incidents.refreshing' : 'incidents.loading')}
          </p> : null}
          {hasLoaded ? <IncidentList
            incidents={incidents}
            statusFilter={statusFilter}
            onStatusFilterChange={changeStatus}
            onSelect={selectIncident}
            selectedId={selectedId}
          /> : null}
        </div>
        <section
          ref={detailRef}
          data-incident-detail
          tabIndex={-1}
          aria-label={t('incidents.detail_label')}
          className={`min-w-0 space-y-3 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent ${linkedIncidentId ? '' : 'hidden xl:block'}`}
        >
          {linkedIncidentId ? <div className="xl:hidden">
            <Button variant="secondary" onClick={() => setSearchParams((previous) => {
              const next = new URLSearchParams(previous);
              next.delete('incident_id');
              return next;
            })}>{t('incidents.back_to_list')}</Button>
          </div> : null}
          {actionError ? <Banner variant="error">{actionError}</Banner> : null}
          {detailError ? <div className="space-y-2">
            <Banner variant="error">{detailError}</Banner>
            <Button variant="secondary" onClick={() => setDetailRetry((retry) => retry + 1)}>{t('incidents.retry')}</Button>
          </div> : detailLoading || (selectedId && !visibleIncident) ? (
            <p role="status" className="min-w-0 rounded-md border border-dashed border-zinc-200 p-6 text-sm text-zinc-600">
              {t('incidents.loading_detail')}
            </p>
          ) : visibleIncident ? (
            <IncidentDetail
              key={visibleIncident.id}
              incident={visibleIncident}
              teams={handoffTargets.map((team) => ({ id: team.id, name: team.name, supportTier: team.support_tier }))}
              owningTeamName={owningTeamName}
              owningTier={owningTier}
              canBounce={canBounce}
              onAcknowledge={(id) => runAction(() => acknowledgeIncident(id), id)}
              onResolve={(id) => runAction(() => resolveIncident(id), id)}
              onHandoff={(id, toTeamId, note) => runAction(() => handoffIncident(id, toTeamId, note), id)}
              onBounce={(id, note) => runAction(() => bounceIncident(id, note), id)}
            />
          ) : (
            <p className="min-w-0 rounded-md border border-dashed border-zinc-200 p-6 text-sm text-zinc-600">
              {t('incidents.select_prompt')}
            </p>
          )}
        </section>
      </div>
    </PageContent>
  );
}
