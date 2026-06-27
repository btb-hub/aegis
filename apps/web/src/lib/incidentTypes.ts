export type IncidentStatus = 'open' | 'acknowledged' | 'resolved';

export type IncidentAlert = {
  id: string;
  severity: string;
  title: string;
  status: string;
};

export type TimelineEvent = {
  id: string;
  kind: string;
  payload: Record<string, string>;
  createdAt: string;
};

export type Incident = {
  id: string;
  teamId: string;
  status: IncidentStatus;
  severity: string;
  title: string;
  fingerprint: string;
  jiraIssueKey?: string;
  createdAt: string;
  acknowledgedAt?: string;
  resolvedAt?: string;
  alerts: IncidentAlert[];
  timeline: TimelineEvent[];
};
