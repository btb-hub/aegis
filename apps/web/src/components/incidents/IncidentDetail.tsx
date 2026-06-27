import { useTranslation } from 'react-i18next';
import { Button } from '../ui/Button';
import { SeverityTag } from '../ui/SeverityTag';
import type { Incident } from '../../lib/incidentTypes';

type IncidentDetailProps = {
  incident: Incident;
  onAcknowledge: (incidentId: string) => void;
  onResolve: (incidentId: string) => void;
};

export function IncidentDetail({ incident, onAcknowledge, onResolve }: IncidentDetailProps) {
  const { t } = useTranslation();
  const canAcknowledge = incident.status === 'open';
  const canResolve = incident.status === 'open' || incident.status === 'acknowledged';

  return (
    <div className="space-y-6 rounded-md border border-zinc-200 bg-white p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <SeverityTag severity="neutral" label={incident.severity} />
            <h2 className="text-2xl font-semibold">{incident.title}</h2>
          </div>
          <p className="text-sm text-zinc-600">
            {t(`incidents.status.${incident.status}`)} · {incident.id}
          </p>
          {incident.jiraIssueKey ? (
            <a
              className="text-sm font-medium text-blue-700 hover:underline"
              href={`https://jira.example.com/browse/${incident.jiraIssueKey}`}
            >
              {t('incidents.jira_link', { key: incident.jiraIssueKey })}
            </a>
          ) : null}
        </div>
        <div className="flex gap-2">
          {canAcknowledge ? (
            <Button onClick={() => onAcknowledge(incident.id)}>{t('incidents.acknowledge')}</Button>
          ) : null}
          {canResolve ? (
            <Button variant="secondary" onClick={() => onResolve(incident.id)}>
              {t('incidents.resolve')}
            </Button>
          ) : null}
        </div>
      </div>

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
                  <SeverityTag severity="neutral" label={alert.severity} />
                  <span>{alert.title}</span>
                </div>
                <span className="capitalize text-zinc-600">{alert.status}</span>
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
                <span className="font-medium capitalize">{event.kind.replaceAll('_', ' ')}</span>
                <time className="text-xs text-zinc-500">{new Date(event.createdAt).toLocaleString()}</time>
              </div>
            </li>
          ))}
        </ol>
      </section>
    </div>
  );
}
