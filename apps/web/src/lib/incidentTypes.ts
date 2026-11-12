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

// INC-NUM-UI: Show the API's display number in list and detail (including its
// accessible label), falling back to the first eight UUID characters when absent.
// Keep id as the UUID for selection and actions. Evidence: incidentsApi.test.ts,
// IncidentList.test.tsx and IncidentDetail.test.tsx.
export type Incident = {
  id: string;
  displayId?: string;
  teamId: string;
  status: IncidentStatus;
  severity: string;
  title: string;
  fingerprint: string;
  jiraIssueKey?: string;
  jiraIssueUrl?: string;
  createdAt: string;
  acknowledgedAt?: string;
  resolvedAt?: string;
  assignee?: IncidentAssignee;
  alerts: IncidentAlert[];
  timeline: TimelineEvent[];
};
