import { apiFetch } from './apiClient';

export type IntegrationItem = {
  id: string;
  kind: string;
  name: string;
  enabled: boolean;
  workspace_id?: string | null;
  config?: Record<string, unknown>;
  config_complete?: boolean;
  mode?: string;
  slot_status?: string;
};

export async function fetchIntegrations(): Promise<IntegrationItem[]> {
  const data = await apiFetch<{ items: IntegrationItem[] }>('/api/v1/integrations');
  return data.items ?? [];
}

export async function createIntegration(payload: Record<string, unknown>): Promise<IntegrationItem> {
  return apiFetch<IntegrationItem>('/api/v1/integrations', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function updateIntegration(
  id: string,
  payload: Record<string, unknown>,
): Promise<IntegrationItem> {
  return apiFetch<IntegrationItem>(`/api/v1/integrations/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function deleteIntegration(id: string): Promise<void> {
  await apiFetch(`/api/v1/integrations/${id}`, { method: 'DELETE' });
}

export async function testIntegration(id: string): Promise<void> {
  await apiFetch(`/api/v1/integrations/${id}/test`, { method: 'POST' });
}
