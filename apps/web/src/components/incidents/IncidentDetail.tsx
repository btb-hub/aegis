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

type IncidentDetailProps = {
  incident: Incident;
  teams: HandoffTeamOption[];
  owningTeamName?: string;
  owningTier?: string;
  canBounce: boolean;
  onAcknowledge: (incidentId: string) => void;
  onResolve: (incidentId: string) => void;
  onHandoff: (incidentId: string, toTeamId: string, note: string) => void;
  onBounce: (incidentId: string, note: string) => void;
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
        className={`${expanded ? '' : 'line-clamp-4'} break-words text-xl font-semibold leading-7 text-zinc-900 [overflow-wrap:anywhere] sm:text-2xl sm:leading-8`}
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
  const [bounceNote, setBounceNote] = useState('');

  useEffect(() => {
    setTargetTeamId(teams[0]?.id ?? '');
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
    <div className="min-w-0 space-y-6 rounded-lg border border-zinc-200 bg-white p-4 sm:p-6">
      <header className="grid min-w-0 gap-4 border-b border-zinc-100 pb-5 lg:grid-cols-[minmax(0,1fr)_auto]">
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
        <div className="flex flex-wrap items-start gap-2 lg:max-w-64 lg:justify-end">
          {canAcknowledge ? (
            <Button onClick={() => onAcknowledge(incident.id)}>{t('incidents.acknowledge')}</Button>
          ) : null}
          {canResolve ? (
            <Button variant="secondary" onClick={() => onResolve(incident.id)}>
              {t('incidents.resolve')}
            </Button>
          ) : null}
          {canHandoff ? (
            <Button variant="secondary" onClick={() => setShowHandoff((open) => !open)}>
              {handoffLabel}
            </Button>
          ) : null}
          {canBounce && incident.status !== 'resolved' ? (
            <Button variant="secondary" onClick={() => setShowBounce((open) => !open)}>
              {bounceLabel}
            </Button>
          ) : null}
        </div>
      </header>

      {showHandoffUnavailable ? (
        <p className="text-sm text-zinc-600">{t('incidents.no_handoff_targets')}</p>
      ) : null}

      {showHandoff ? (
        <section className="min-w-0 space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-4">
          <h3 className="text-sm font-semibold">{handoffHeading}</h3>
          <Select
            id="handoff-target-team"
            label={handoffTeamLabel}
            value={targetTeamId}
            options={teamOptions}
            onChange={setTargetTeamId}
          />
          <Input
            label={t('incidents.handoff_note_label')}
            value={handoffNote}
            onChange={setHandoffNote}
          />
          <Button
            onClick={() => {
              onHandoff(incident.id, targetTeamId, handoffNote);
              setShowHandoff(false);
              setHandoffNote('');
            }}
          >
            {t('incidents.handoff_submit')}
          </Button>
        </section>
      ) : null}

      {showBounce ? (
        <section className="min-w-0 space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-4">
          <h3 className="text-sm font-semibold">{bounceHeading}</h3>
          <Input
            label={t('incidents.bounce_note_label')}
            value={bounceNote}
            onChange={setBounceNote}
          />
          <Button
            onClick={() => {
              onBounce(incident.id, bounceNote);
              setShowBounce(false);
              setBounceNote('');
            }}
          >
            {t('incidents.bounce_submit')}
          </Button>
        </section>
      ) : null}

      <section className="min-w-0 space-y-2 rounded-md bg-zinc-50 p-4">
        <h3 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
          {t('incidents.assignee_heading')}
        </h3>
        {incident.assignee ? (
          <PersonContacts displayName={incident.assignee.displayName} contacts={incident.assignee.contacts} />
        ) : (
          <p className="text-sm text-zinc-600">{t('incidents.no_assignee')}</p>
        )}
      </section>

      <section className="space-y-2">
        <h3 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
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
        <h3 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
          {t('incidents.timeline_heading')}
        </h3>
        <ol className="space-y-3">
          {incident.timeline.map((event) => (
            <li key={event.id} className="rounded-md border border-zinc-200 px-4 py-3">
              <div className="flex min-w-0 flex-col gap-1 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
                <span className="min-w-0 break-words font-medium [overflow-wrap:anywhere]">
                  {event.payload.message ??
                    t(`incidents.timeline.${event.kind}`, { defaultValue: event.kind })}
                </span>
                <time className="shrink-0 text-xs text-zinc-500">
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
