import type { Incident, IncidentAlert, TimelineEvent } from './incidentTypes';

export type ApiIncident = {
  id: string;
  team_id: string;
  status: 'open' | 'acknowledged' | 'resolved';
  severity: string;
  title: string;
  fingerprint: string;
  jira_issue_key?: string;
  created_at: string;
  acknowledged_at?: string;
  resolved_at?: string;
  assignee_id?: string;
};

export type ApiTimelineEvent = {
  id: string;
  kind: string;
  payload: Record<string, unknown>;
  created_at: string;
  actor_id?: string;
};

export type ApiAlertSummary = {
  id: string;
  severity: string;
  title: string;
  status: string;
};

async function parseJson<T>(response: Response): Promise<T> {
  return (await response.json()) as T;
}

async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, { credentials: 'include', ...init });
  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return parseJson<T>(response);
}

function mapTimelinePayload(payload: Record<string, unknown>): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [key, value] of Object.entries(payload)) {
    if (value == null) {
      continue;
    }
    out[key] = String(value);
  }
  return out;
}

export function mapApiTimelineEvent(event: ApiTimelineEvent): TimelineEvent {
  return {
    id: event.id,
    kind: event.kind,
    payload: mapTimelinePayload(event.payload ?? {}),
    createdAt: event.created_at,
  };
}

export function mapApiIncident(
  incident: ApiIncident,
  alerts: IncidentAlert[] = [],
  timeline: TimelineEvent[] = [],
): Incident {
  return {
    id: incident.id,
    teamId: incident.team_id,
    status: incident.status,
    severity: incident.severity,
    title: incident.title,
    fingerprint: incident.fingerprint,
    jiraIssueKey: incident.jira_issue_key,
    createdAt: incident.created_at,
    acknowledgedAt: incident.acknowledged_at,
    resolvedAt: incident.resolved_at,
    alerts,
    timeline,
  };
}

export async function fetchIncidents(status?: string): Promise<Incident[]> {
  const query = status && status !== 'all' ? `?status=${encodeURIComponent(status)}` : '';
  const data = await apiFetch<{ items: ApiIncident[] }>(`/api/v1/incidents${query}`);
  return (data.items ?? []).map((item) => mapApiIncident(item));
}

export async function fetchIncidentDetail(id: string): Promise<Incident> {
  const [detail, timelineData] = await Promise.all([
    apiFetch<{ incident: ApiIncident; alerts: ApiAlertSummary[] }>(`/api/v1/incidents/${id}`),
    apiFetch<{ items: ApiTimelineEvent[] }>(`/api/v1/incidents/${id}/timeline`),
  ]);
  const alerts: IncidentAlert[] = (detail.alerts ?? []).map((alert) => ({
    id: alert.id,
    severity: alert.severity,
    title: alert.title,
    status: alert.status,
  }));
  const timeline = (timelineData.items ?? []).map(mapApiTimelineEvent);
  return mapApiIncident(detail.incident, alerts, timeline);
}

export async function acknowledgeIncident(id: string): Promise<void> {
  await apiFetch(`/api/v1/incidents/${id}/acknowledge`, { method: 'POST' });
}

export async function resolveIncident(id: string): Promise<void> {
  await apiFetch(`/api/v1/incidents/${id}/resolve`, { method: 'POST' });
}

export async function handoffIncident(id: string, toTeamId: string, note: string): Promise<void> {
  await apiFetch(`/api/v1/incidents/${id}/handoff`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ to_team_id: toTeamId, note }),
  });
}

export async function bounceIncident(id: string, note: string): Promise<void> {
  await apiFetch(`/api/v1/incidents/${id}/bounce`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ note }),
  });
}

export type HandoffTarget = {
  id: string;
  name: string;
  support_tier?: string;
};

export async function fetchHandoffTargets(teamId: string): Promise<HandoffTarget[]> {
  const data = await apiFetch<{ items: HandoffTarget[] }>(`/api/v1/teams/${teamId}/handoff-targets`);
  return data.items ?? [];
}
