import type { Workspace } from './teamTypes';

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
  team_id: string;
  match_labels: Record<string, string>;
  priority: number;
  created_at: string;
  updated_at: string;
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

export async function fetchWorkspaces(): Promise<Workspace[]> {
  const data = await apiFetch<{ items: Workspace[] }>('/api/v1/workspaces');
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
  team_id: string;
  match_labels: Record<string, string>;
  priority: number;
}): Promise<RoutingRule> {
  return apiFetch<RoutingRule>('/api/v1/routing-rules', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function updateRoutingRule(
  id: string,
  payload: { team_id: string; match_labels: Record<string, string>; priority: number },
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
