import type { Team, Workspace } from './teamTypes';

export type WorkspaceSummary = Workspace & {
  team_count: number;
  routing_rule_count: number;
};

export type EscalationPath = {
  id: string;
  from_team_id: string;
  to_team_id: string;
  workspace_id: string;
  cross_workspace: boolean;
  created_at: string;
};

export type RoutingRule = {
  id: string;
  workspace_id: string;
  team_id: string;
  match_labels: Record<string, string>;
  priority: number;
  cross_workspace: boolean;
  created_at: string;
  updated_at: string;
};

export type BlockedTeamMove = {
  team_id: string;
  paths: EscalationPath[];
};

export class WorkspaceApiError extends Error {
  status: number;
  code?: string;
  details?: { blocked_teams?: BlockedTeamMove[] };

  constructor(message: string, status: number, code?: string, details?: WorkspaceApiError['details']) {
    super(message);
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

async function parseJson<T>(response: Response): Promise<T> {
  return (await response.json()) as T;
}

async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, { credentials: 'include', ...init });
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as {
      message?: string;
      code?: string;
      details?: WorkspaceApiError['details'];
    };
    throw new WorkspaceApiError(
      body.message ?? `request failed: ${response.status}`,
      response.status,
      body.code,
      body.details,
    );
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return parseJson<T>(response);
}

export async function fetchWorkspaces(): Promise<WorkspaceSummary[]> {
  const data = await apiFetch<{ items: WorkspaceSummary[] }>('/api/v1/workspaces');
  return data.items ?? [];
}

export async function fetchWorkspace(id: string): Promise<Workspace> {
  return apiFetch<Workspace>(`/api/v1/workspaces/${id}`);
}

export async function createWorkspace(payload: {
  name: string;
  slug?: string;
  description?: string;
}): Promise<Workspace> {
  return apiFetch<Workspace>('/api/v1/workspaces', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function updateWorkspace(
  id: string,
  payload: { name: string; slug?: string; description?: string },
): Promise<Workspace> {
  return apiFetch<Workspace>(`/api/v1/workspaces/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function deleteWorkspace(id: string): Promise<void> {
  await apiFetch(`/api/v1/workspaces/${id}`, { method: 'DELETE' });
}

export async function assignTeamsToWorkspace(workspaceId: string, teamIds: string[]): Promise<Team[]> {
  const data = await apiFetch<{ items: Team[] }>(`/api/v1/workspaces/${workspaceId}/teams`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ team_ids: teamIds }),
  });
  return data.items ?? [];
}

export async function fetchEscalationPaths(workspaceId: string): Promise<EscalationPath[]> {
  const data = await apiFetch<{ items: EscalationPath[] }>(
    `/api/v1/workspaces/${workspaceId}/escalation-paths`,
  );
  return data.items ?? [];
}

export async function addEscalationPath(
  workspaceId: string,
  payload: { from_team_id: string; to_team_id: string; cross_workspace?: boolean },
): Promise<EscalationPath> {
  return apiFetch<EscalationPath>(`/api/v1/workspaces/${workspaceId}/escalation-paths`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function deleteEscalationPath(pathId: string): Promise<void> {
  await apiFetch(`/api/v1/escalation-paths/${pathId}`, { method: 'DELETE' });
}

export async function fetchOutgoingPaths(teamId: string): Promise<EscalationPath[]> {
  const data = await apiFetch<{ items: EscalationPath[] }>(
    `/api/v1/teams/${teamId}/escalation-paths/outgoing`,
  );
  return data.items ?? [];
}

export async function fetchIncomingPaths(teamId: string): Promise<EscalationPath[]> {
  const data = await apiFetch<{ items: EscalationPath[] }>(
    `/api/v1/teams/${teamId}/escalation-paths/incoming`,
  );
  return data.items ?? [];
}

export async function fetchRoutingRules(): Promise<RoutingRule[]> {
  const data = await apiFetch<{ items: RoutingRule[] }>('/api/v1/routing-rules');
  return data.items ?? [];
}

export async function createRoutingRule(payload: {
  workspace_id: string;
  team_id: string;
  match_labels: Record<string, string>;
  priority: number;
  cross_workspace: boolean;
}): Promise<RoutingRule> {
  return apiFetch<RoutingRule>('/api/v1/routing-rules', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function updateRoutingRule(
  id: string,
  payload: {
    workspace_id: string;
    team_id: string;
    match_labels: Record<string, string>;
    priority: number;
    cross_workspace: boolean;
  },
): Promise<RoutingRule> {
  return apiFetch<RoutingRule>(`/api/v1/routing-rules/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function deleteRoutingRule(id: string): Promise<void> {
  await apiFetch(`/api/v1/routing-rules/${id}`, { method: 'DELETE' });
}
