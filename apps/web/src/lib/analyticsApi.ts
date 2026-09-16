import { apiFetch } from './apiClient';
import { defaultAnalyticsRange, type OverviewAnalytics } from './analyticsTypes';

export async function fetchOverview(comparePrevious: boolean): Promise<OverviewAnalytics> {
  const { from, to } = defaultAnalyticsRange();
  const params = new URLSearchParams({
    from,
    to,
    compare_previous: comparePrevious ? 'true' : 'false',
  });
  return apiFetch<OverviewAnalytics>(`/api/v1/analytics/overview?${params.toString()}`);
}
