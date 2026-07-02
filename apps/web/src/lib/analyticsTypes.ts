export type MetricBucket = {
  bucket_start: string;
  mean_seconds: number;
  count: number;
};

export type MetricPeriod = {
  from: string;
  to: string;
  mean_seconds: number;
  count: number;
  series: MetricBucket[];
};

export type MetricAnalytics = {
  from: string;
  to: string;
  mean_seconds: number;
  count: number;
  series: MetricBucket[];
  previous?: MetricPeriod;
};

export type NoiseItem = {
  fingerprint: string;
  title: string;
  count: number;
};

export type OnCallLoadItem = {
  user_id: string;
  display_name: string;
  email: string;
  page_count: number;
};

export type HandoffStats = {
  count: number;
  median_response_seconds: number;
};

export type EscalationStats = {
  total_incidents: number;
  escalated_count: number;
  escalated_percent: number;
  mean_seconds_to_escalate: number;
};

export type OverviewAnalytics = {
  from: string;
  to: string;
  mtta: MetricAnalytics;
  mttr: MetricAnalytics;
  noise: { items: NoiseItem[] };
  on_call_load: { items: OnCallLoadItem[] };
  handoffs: HandoffStats;
  escalation: EscalationStats;
};

export function defaultAnalyticsRange(): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to);
  from.setDate(from.getDate() - 7);
  return { from: from.toISOString(), to: to.toISOString() };
}

export function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return `${Math.round(seconds)}s`;
  }
  if (seconds < 3600) {
    return `${Math.round(seconds / 60)}m`;
  }
  return `${(seconds / 3600).toFixed(1)}h`;
}
