import { useTranslation } from 'react-i18next';
import { formatDateTime } from '../../lib/formatDate';
import type { Incident } from '../../lib/incidentTypes';
import { severityLabelKey, severityToTag } from '../../lib/severityTag';
import { Button } from '../ui/Button';
import { SeverityTag } from '../ui/SeverityTag';
import { StatusTag, alertStatusVariant, incidentStatusVariant } from '../ui/StatusTag';

type IncidentDetailProps = {
  incident: Incident;
  onAcknowledge: (incidentId: string) => void;
  onResolve: (incidentId: string) => void;
};

export function IncidentDetail({ incident, onAcknowledge, onResolve }: IncidentDetailProps) {
  const { t, i18n } = useTranslation();
  const canAcknowledge = incident.status === 'open';
  const canResolve = incident.status === 'open' || incident.status === 'acknowledged';

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
