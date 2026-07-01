export type AlertStatus = 'firing' | 'resolved';

export type AlertItem = {
  id: string;
  fingerprint: string;
  status: AlertStatus;
  severity: string;
  title: string;
  body?: string;
  labels: Record<string, string>;
  received_at: string;
};

export type AlertGroup = {
  key: string;
  count: number;
  sample?: AlertItem;
};

export type AlertAnalytics = {
  by_severity: Record<string, number>;
  by_status: Record<string, number>;
  top_labels: Array<{ key: string; value: string; count: number }>;
};

export type AlertFilters = {
  q: string;
  severity: string;
  status: string;
  teamId: string;
  from: string;
  to: string;
  labelKey: string;
  labelValue: string;
  groupBy: '' | 'severity' | 'label';
  groupLabelKey: string;
};

export type SavedView = {
  id: string;
  owner_id: string;
  name: string;
  filter: Record<string, unknown>;
  shared: boolean;
};

export const defaultAlertFilters = (): AlertFilters => ({
  q: '',
  severity: '',
  status: '',
  teamId: '',
  from: '',
  to: '',
  labelKey: '',
  labelValue: '',
  groupBy: '',
  groupLabelKey: 'team',
});

export function filtersToQuery(filters: AlertFilters, page: number, pageSize: number): URLSearchParams {
  const params = new URLSearchParams();
  if (filters.q) {
    params.set('q', filters.q);
  }
  if (filters.severity) {
    params.set('severity', filters.severity);
  }
  if (filters.status) {
    params.set('status', filters.status);
  }
  if (filters.teamId) {
    params.set('team_id', filters.teamId);
  }
  if (filters.from) {
    params.set('from', filters.from);
  }
  if (filters.to) {
    params.set('to', filters.to);
  }
  if (filters.labelKey) {
    params.set('label', `${filters.labelKey}:${filters.labelValue}`);
  }
  if (filters.groupBy === 'severity') {
    params.set('group_by', 'severity');
  } else if (filters.groupBy === 'label' && filters.groupLabelKey) {
    params.set('group_by', `label:${filters.groupLabelKey}`);
  }
  params.set('page', String(page));
  params.set('page_size', String(pageSize));
  params.set('include_analytics', 'true');
  if (filters.groupLabelKey) {
    params.set('analytics_label_key', filters.groupLabelKey);
  }
  return params;
}

export function filtersToExportQuery(filters: AlertFilters): URLSearchParams {
  const params = filtersToQuery(filters, 1, 100);
  params.delete('page');
  params.delete('page_size');
  params.delete('include_analytics');
  params.delete('analytics_label_key');
  params.delete('group_by');
  return params;
}

export function filtersToSavedView(filters: AlertFilters): Record<string, unknown> {
  const view: Record<string, unknown> = {};
  if (filters.q) {
    view.q = filters.q;
  }
  if (filters.severity) {
    view.severity = filters.severity;
  }
  if (filters.status) {
    view.status = filters.status;
  }
  if (filters.teamId) {
    view.team_id = filters.teamId;
  }
  if (filters.from) {
    view.from = filters.from;
  }
  if (filters.to) {
    view.to = filters.to;
  }
  if (filters.labelKey) {
    view.label_key = filters.labelKey;
    view.label_value = filters.labelValue;
  }
  if (filters.groupBy) {
    view.group_by = filters.groupBy;
    view.group_label_key = filters.groupLabelKey;
  }
  return view;
}

export function savedViewToFilters(filter: Record<string, unknown>): AlertFilters {
  const next = defaultAlertFilters();
  if (typeof filter.q === 'string') {
    next.q = filter.q;
  }
  if (typeof filter.severity === 'string') {
    next.severity = filter.severity;
  }
  if (typeof filter.status === 'string') {
    next.status = filter.status;
  }
  if (typeof filter.team_id === 'string') {
    next.teamId = filter.team_id;
  }
  if (typeof filter.from === 'string') {
    next.from = filter.from;
  }
  if (typeof filter.to === 'string') {
    next.to = filter.to;
  }
  if (typeof filter.label_key === 'string') {
    next.labelKey = filter.label_key;
    next.labelValue = typeof filter.label_value === 'string' ? filter.label_value : '';
  }
  if (filter.group_by === 'severity') {
    next.groupBy = 'severity';
  } else if (filter.group_by === 'label') {
    next.groupBy = 'label';
    if (typeof filter.group_label_key === 'string') {
      next.groupLabelKey = filter.group_label_key;
    }
  }
  return next;
}
