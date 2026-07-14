import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { formatDateTime } from '../../lib/formatDate';
import type { Incident } from '../../lib/incidentTypes';
import { bounceLabelKey, handoffLabelKey, handoffTeamLabelKey } from '../../lib/teamTypes';
import { severityLabelKey, severityToTag } from '../../lib/severityTag';
import { Button } from '../ui/Button';
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
    <div className="space-y-6 rounded-md border border-zinc-200 bg-white p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-2">
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
            <h2 className="text-2xl font-semibold">{incident.title}</h2>
          </div>
          <p className="text-sm text-zinc-600">{incident.id}</p>
          {incident.jiraIssueKey ? (
            <a
              className="text-sm font-medium text-blue-700 hover:underline"
              href={`https://jira.example.com/browse/${incident.jiraIssueKey}`}
            >
              {t('incidents.jira_link', { key: incident.jiraIssueKey })}
            </a>
          ) : null}
        </div>
        <div className="flex flex-wrap gap-2">
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
      </div>

      {showHandoffUnavailable ? (
        <p className="text-sm text-zinc-600">{t('incidents.no_handoff_targets')}</p>
      ) : null}

      {showHandoff ? (
        <section className="space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-4">
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
        <section className="space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-4">
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

      <section className="space-y-2">
        <h3 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
          {t('incidents.alerts_heading')}
        </h3>
        {incident.alerts.length === 0 ? (
          <p className="text-sm text-zinc-600">{t('incidents.alerts_empty')}</p>
        ) : (
          <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200">
            {incident.alerts.map((alert) => (
              <li key={alert.id} className="flex items-center justify-between px-4 py-3 text-sm">
                <div className="flex items-center gap-2">
                  <SeverityTag
                    severity={severityToTag(alert.severity)}
                    label={t(severityLabelKey(alert.severity), { defaultValue: alert.severity })}
                  />
                  <span>{alert.title}</span>
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
              <div className="flex items-center justify-between gap-4">
                <span className="font-medium">
                  {t(`incidents.timeline.${event.kind}`, { defaultValue: event.kind })}
                </span>
                <time className="text-xs text-zinc-500">
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
