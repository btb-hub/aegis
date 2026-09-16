import { apiFetch, ApiError } from './apiClient';
import type { AlertAnalytics, AlertFilters, AlertGroup, AlertItem, SavedView } from './alertTypes';
import { filtersToExportQuery, filtersToQuery } from './alertTypes';

export type AlertListResponse = {
  items?: AlertItem[];
  groups?: AlertGroup[];
  group_by?: string;
  total?: number;
  analytics?: AlertAnalytics;
};

export async function fetchAlerts(
  filters: AlertFilters,
  page: number,
  pageSize: number,
): Promise<AlertListResponse> {
  const params = filtersToQuery(filters, page, pageSize);
  return apiFetch<AlertListResponse>(`/api/v1/alerts?${params.toString()}`);
}

export async function fetchSavedViews(): Promise<SavedView[]> {
  const data = await apiFetch<{ items: SavedView[] }>('/api/v1/saved-views');
  return data.items ?? [];
}

export async function createSavedView(payload: {
  name: string;
  filter: Record<string, unknown>;
  shared: boolean;
}): Promise<SavedView> {
  return apiFetch<SavedView>('/api/v1/saved-views', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function exportAlertsCsv(filters: AlertFilters): Promise<Blob> {
  const params = filtersToExportQuery(filters);
  const response = await fetch(`/api/v1/alerts/export?${params.toString()}`, { credentials: 'include' });
  if (!response.ok) {
    throw new ApiError('export failed', response.status);
  }
  return response.blob();
}
