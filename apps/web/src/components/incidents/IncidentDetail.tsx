import { useEffect, useId, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { formatDateTime } from '../../lib/formatDate';
import type { Incident } from '../../lib/incidentTypes';
import { splitIncidentTitle } from '../../lib/incidentTitle';
import { bounceLabelKey, handoffLabelKey, handoffTeamLabelKey } from '../../lib/teamTypes';
import { severityLabelKey, severityToTag } from '../../lib/severityTag';
import { Button } from '../ui/Button';
import { PersonContacts } from '../ui/PersonContacts';
import { Input } from '../ui/Input';
import { Select } from '../ui/Select';
import { SeverityTag } from '../ui/SeverityTag';
import { StatusTag, alertStatusVariant, incidentStatusVariant } from '../ui/StatusTag';

export type HandoffTeamOption = {
  id: string;
  name: string;
  supportTier?: string;
};

type IncidentAction = () => void | boolean | Promise<void | boolean>;

type IncidentDetailProps = {
  incident: Incident;
  teams: HandoffTeamOption[];
  owningTeamName?: string;
  owningTier?: string;
  canBounce: boolean;
  onAcknowledge: (incidentId: string) => ReturnType<IncidentAction>;
  onResolve: (incidentId: string) => ReturnType<IncidentAction>;
  onHandoff: (incidentId: string, toTeamId: string, note: string) => ReturnType<IncidentAction>;
  onBounce: (incidentId: string, note: string) => ReturnType<IncidentAction>;
};

function IncidentHeading({ title }: { title: string }) {
  const { t } = useTranslation();
  const titleId = useId();
  const titleRef = useRef<HTMLHeadingElement>(null);
  const { summary, context } = useMemo(() => splitIncidentTitle(title), [title]);
  const [expanded, setExpanded] = useState(false);
  const [canExpand, setCanExpand] = useState(summary.length > 180);

  useEffect(() => {
    setExpanded(false);
  }, [title]);

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
  }, [expanded, summary]);

  return (
    <div className="min-w-0 space-y-2">
      <h2
        ref={titleRef}
        id={titleId}
        tabIndex={-1}
        className={`${expanded ? '' : 'line-clamp-4'} break-words text-xl font-semibold leading-7 text-zinc-900 [overflow-wrap:anywhere]`}
      >
        {summary}
      </h2>
      {canExpand ? (
        <button
          type="button"
          className="rounded-sm text-xs font-medium text-accent hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
          aria-controls={titleId}
          aria-expanded={expanded}
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? t('incidents.title.show_less') : t('incidents.title.show_full')}
        </button>
      ) : null}
      {context ? (
        <details className="rounded-md border border-zinc-200 bg-zinc-50 px-3 py-2 text-sm text-zinc-700">
          <summary className="cursor-pointer font-medium text-zinc-800 marker:text-zinc-400">
            {t('incidents.alert_context')}
          </summary>
          <p className="mt-2 break-words font-mono text-xs leading-5 text-zinc-600 [overflow-wrap:anywhere]">
            {context}
          </p>
        </details>
      ) : null}
    </div>
  );
}

export function IncidentDetail({
  incident,
  teams,
  owningTeamName,
  owningTier,
  canBounce,
  onAcknowledge,
  onResolve,
  onHandoff,
  onBounce,
}: IncidentDetailProps) {
  const { t, i18n } = useTranslation();
  const [showHandoff, setShowHandoff] = useState(false);
  const [showBounce, setShowBounce] = useState(false);
  const [targetTeamId, setTargetTeamId] = useState(teams[0]?.id ?? '');
  const [handoffNote, setHandoffNote] = useState('');
  const [pending, setPending] = useState(false);
  const [feedback, setFeedback] = useState<'success' | 'error' | null>(null);
  const [bounceNote, setBounceNote] = useState('');
  const surfaceRef = useRef<HTMLDivElement>(null);
  const targetIncidentId = useRef(incident.id);

  useEffect(() => {
    const sameIncident = targetIncidentId.current === incident.id;
    targetIncidentId.current = incident.id;
    setTargetTeamId((current) => sameIncident && teams.some((team) => team.id === current)
      ? current : teams[0]?.id ?? '');
  }, [teams, incident.id]);

  const teamOptions = useMemo(
    () =>
      teams.map((team) => ({
        value: team.id,
        label: team.supportTier
          ? `${team.name} (${t(`teams.tier.${team.supportTier}`, { defaultValue: team.supportTier.toUpperCase() })})`
          : team.name,
      })),
    [teams, t],
  );

  useEffect(() => {
    setShowHandoff(false);
    setShowBounce(false);
    setHandoffNote('');
    setBounceNote('');
    setFeedback(null);
  }, [incident.id]);

  const perform = async (action: IncidentAction, complete?: () => void) => {
    if (pending) {
      return;
    }
    const focused = document.activeElement;
    const startedInDetail = focused instanceof HTMLElement && surfaceRef.current?.contains(focused);
    setPending(true);
    setFeedback(null);
    try {
      if (await action() !== false) {
        complete?.();
        setFeedback('success');
        if (startedInDetail && (surfaceRef.current?.contains(document.activeElement)
          || (document.activeElement === document.body && !focused.isConnected))) {
          surfaceRef.current?.querySelector<HTMLElement>('h2')?.focus();
        }
      }
    } catch {
      setFeedback('error');
    } finally {
      setPending(false);
    }
  };

  const handoffLabel = t(handoffLabelKey(owningTier));
  const bounceLabel = t(bounceLabelKey(owningTier));
  const handoffHeading = t(`${handoffLabelKey(owningTier)}_heading`, { defaultValue: handoffLabel });
  const bounceHeading = t(`${bounceLabelKey(owningTier)}_heading`, { defaultValue: bounceLabel });
  const handoffTeamLabel = t(handoffTeamLabelKey(owningTier));

  const canAcknowledge = incident.status === 'open';
  const canResolve = incident.status === 'open' || incident.status === 'acknowledged';
  const canHandoff = incident.status !== 'resolved' && teams.length > 0;
  const showHandoffUnavailable = incident.status !== 'resolved' && teams.length === 0;

  return (
    <div ref={surfaceRef} className="min-w-0 space-y-6 rounded-lg border border-zinc-200 bg-white p-4 sm:p-5">
      <header className="min-w-0 space-y-4 border-b border-zinc-200 pb-4">
        <div className="min-w-0 space-y-3">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <SeverityTag
              severity={severityToTag(incident.severity)}
              label={t(severityLabelKey(incident.severity), { defaultValue: incident.severity })}
            />
            <StatusTag
              variant={incidentStatusVariant(incident.status)}
              label={t(`incidents.status.${incident.status}`)}
            />
            {owningTeamName ? (
              <span className="text-sm text-zinc-700">
                {t('incidents.owning_team', { name: owningTeamName })}
              </span>
            ) : null}
            {owningTier ? (
              <StatusTag
                variant="neutral"
                label={t(`teams.tier.${owningTier}`, { defaultValue: owningTier.toUpperCase() })}
              />
            ) : null}
          </div>
          <IncidentHeading title={incident.title} />
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-zinc-500">
            <span className="font-mono" aria-label={t('incidents.id_label', { id: incident.id })}>
              {incident.id.slice(0, 8)}
            </span>
            <time dateTime={incident.createdAt}>
              {t('incidents.created_at', {
                value: formatDateTime(new Date(incident.createdAt), i18n.language),
              })}
            </time>
          </div>
          {incident.jiraIssueKey ? (
            <a
              className="text-sm font-medium text-blue-700 hover:underline"
              href={`https://jira.example.com/browse/${incident.jiraIssueKey}`}
            >
              {t('incidents.jira_link', { key: incident.jiraIssueKey })}
            </a>
          ) : null}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {canAcknowledge ? (
            <Button variant={showHandoff || showBounce ? 'secondary' : 'primary'} disabled={pending} onClick={() => void perform(() => onAcknowledge(incident.id))}>{t('incidents.acknowledge')}</Button>
          ) : null}
          {canResolve ? (
            <Button variant={canAcknowledge || showHandoff || showBounce ? 'secondary' : 'primary'} disabled={pending} onClick={() => void perform(() => onResolve(incident.id))}>
              {t('incidents.resolve')}
            </Button>
          ) : null}
          {canHandoff ? (
            <Button variant="secondary" disabled={pending} aria-expanded={showHandoff} onClick={() => setShowHandoff((open) => !open)}>
              {handoffLabel}
            </Button>
          ) : null}
          {canBounce && incident.status !== 'resolved' ? (
            <Button variant="secondary" disabled={pending} aria-expanded={showBounce} onClick={() => setShowBounce((open) => !open)}>
              {bounceLabel}
            </Button>
          ) : null}
        </div>
      </header>

      {pending || feedback ? <p role="status" className={`text-sm ${feedback === 'error' ? 'text-severity-p1' : 'text-zinc-600'}`}>{t(pending ? 'incidents.action_pending' : feedback === 'error' ? 'incidents.action_error' : 'incidents.action_complete')}</p> : null}

      {showHandoffUnavailable ? (
        <p className="text-sm text-zinc-600">{t('incidents.no_handoff_targets')}</p>
      ) : null}

      {showHandoff && canHandoff ? (
        <section className="min-w-0 space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-4">
          <h3 className="text-sm font-semibold">{handoffHeading}</h3>
          <Select
            id="handoff-target-team"
            label={handoffTeamLabel}
            value={targetTeamId}
            options={teamOptions}
            onChange={setTargetTeamId}
            disabled={pending}
          />
          <Input
            label={t('incidents.handoff_note_label')}
            value={handoffNote}
            onChange={setHandoffNote}
            disabled={pending}
          />
          <Button
            disabled={pending || !targetTeamId}
            onClick={() => void perform(() => onHandoff(incident.id, targetTeamId, handoffNote), () => {
              setShowHandoff(false);
              setHandoffNote('');
            })}
          >
            {t('incidents.handoff_submit')}
          </Button>
        </section>
      ) : null}

      {showBounce && canBounce && incident.status !== 'resolved' ? (
        <section className="min-w-0 space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-4">
          <h3 className="text-sm font-semibold">{bounceHeading}</h3>
          <Input
            label={t('incidents.bounce_note_label')}
            value={bounceNote}
            onChange={setBounceNote}
            disabled={pending}
          />
          <Button
            disabled={pending}
            onClick={() => void perform(() => onBounce(incident.id, bounceNote), () => {
              setShowBounce(false);
              setBounceNote('');
            })}
          >
            {t('incidents.bounce_submit')}
          </Button>
        </section>
      ) : null}

      <section className="min-w-0 space-y-2 rounded-md bg-zinc-50 p-4">
        <h3 className="text-sm font-semibold text-zinc-800">
          {t('incidents.assignee_heading')}
        </h3>
        {incident.assignee ? (
          <PersonContacts displayName={incident.assignee.displayName} contacts={incident.assignee.contacts} />
        ) : (
          <p className="text-sm text-zinc-600">{t('incidents.no_assignee')}</p>
        )}
      </section>

      <section className="space-y-2">
        <h3 className="text-sm font-semibold text-zinc-800">
          {t('incidents.alerts_heading')}
        </h3>
        {incident.alerts.length === 0 ? (
          <p className="text-sm text-zinc-600">{t('incidents.alerts_empty')}</p>
        ) : (
          <ul className="min-w-0 divide-y divide-zinc-200 overflow-hidden rounded-md border border-zinc-200">
            {incident.alerts.map((alert) => (
              <li
                key={alert.id}
                className="grid min-w-0 gap-3 px-4 py-3 text-sm sm:grid-cols-[minmax(0,1fr)_auto] sm:items-start"
              >
                <div className="flex min-w-0 items-start gap-2">
                  <SeverityTag
                    severity={severityToTag(alert.severity)}
                    label={t(severityLabelKey(alert.severity), { defaultValue: alert.severity })}
                  />
                  <span className="line-clamp-3 min-w-0 break-words leading-5 text-zinc-800 [overflow-wrap:anywhere]">
                    {splitIncidentTitle(alert.title).summary}
                  </span>
                </div>
                <StatusTag
                  variant={alertStatusVariant(alert.status)}
                  label={t(`incidents.alert_status.${alert.status.toLowerCase()}`, {
                    defaultValue: alert.status,
                  })}
                />
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="space-y-2">
        <h3 className="text-sm font-semibold text-zinc-800">
          {t('incidents.timeline_heading')}
        </h3>
        <ol className="ml-1 border-l border-zinc-200">
          {incident.timeline.map((event) => (
            <li key={event.id} className="relative py-2 pl-5 before:absolute before:-left-1 before:top-4 before:h-2 before:w-2 before:rounded-full before:bg-zinc-400">
              <div className="flex min-w-0 flex-col gap-1 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
                <span className="min-w-0 break-words font-medium [overflow-wrap:anywhere]">
                  {event.payload.message ??
                    t(`incidents.timeline.${event.kind}`, { defaultValue: event.kind })}
                </span>
                <time dateTime={event.createdAt} className="shrink-0 text-xs tabular-nums text-zinc-600">
                  {formatDateTime(new Date(event.createdAt), i18n.language, {
                    second: '2-digit',
                  })}
                </time>
              </div>
            </li>
          ))}
        </ol>
      </section>
    </div>
  );
}
