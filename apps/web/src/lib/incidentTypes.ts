import type { ContactPerson } from './contactTypes';

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

export type IncidentAssignee = ContactPerson;

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
  assignee?: IncidentAssignee;
  alerts: IncidentAlert[];
  timeline: TimelineEvent[];
};
