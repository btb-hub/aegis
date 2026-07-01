import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { formatDateTime } from '../../lib/formatDate';
import type { Incident } from '../../lib/incidentTypes';
import { severityLabelKey, severityToTag } from '../../lib/severityTag';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { SeverityTag } from '../ui/SeverityTag';
import { StatusTag, alertStatusVariant, incidentStatusVariant } from '../ui/StatusTag';

export type HandoffTeamOption = {
  id: string;
  name: string;
};

type IncidentDetailProps = {
  incident: Incident;
  teams: HandoffTeamOption[];
  canBounce: boolean;
  onAcknowledge: (incidentId: string) => void;
  onResolve: (incidentId: string) => void;
  onHandoff: (incidentId: string, toTeamId: string, note: string) => void;
  onBounce: (incidentId: string, note: string) => void;
};

export function IncidentDetail({
  incident,
  teams,
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

  const canAcknowledge = incident.status === 'open';
  const canResolve = incident.status === 'open' || incident.status === 'acknowledged';
  const canHandoff = incident.status !== 'resolved' && teams.length > 0;

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
              {t('incidents.handoff')}
            </Button>
          ) : null}
          {canBounce && incident.status !== 'resolved' ? (
            <Button variant="secondary" onClick={() => setShowBounce((open) => !open)}>
              {t('incidents.bounce')}
            </Button>
          ) : null}
        </div>
      </div>

      {showHandoff ? (
        <section className="space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-4">
          <h3 className="text-sm font-semibold">{t('incidents.handoff_heading')}</h3>
          <label className="block space-y-1 text-sm">
            <span>{t('incidents.handoff_team_label')}</span>
            <select
              aria-label={t('incidents.handoff_team_label')}
              className="w-full rounded-md border border-zinc-300 bg-white px-3 py-2"
              value={targetTeamId}
              onChange={(event) => setTargetTeamId(event.target.value)}
            >
              {teams.map((team) => (
                <option key={team.id} value={team.id}>
                  {team.name}
                </option>
              ))}
            </select>
          </label>
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
          <h3 className="text-sm font-semibold">{t('incidents.bounce_heading')}</h3>
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
