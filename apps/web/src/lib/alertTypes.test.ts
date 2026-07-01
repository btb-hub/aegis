import { describe, expect, it } from 'vitest';
import {
  defaultAlertFilters,
  filtersToExportQuery,
  filtersToQuery,
  filtersToSavedView,
  savedViewToFilters,
} from './alertTypes';

describe('alertTypes helpers', () => {
  it('builds default filters', () => {
    const filters = defaultAlertFilters();
    expect(filters.groupLabelKey).toBe('team');
    expect(filters.q).toBe('');
  });

  it('serializes filters to query params', () => {
    const filters = {
      ...defaultAlertFilters(),
      q: 'cpu',
      severity: 'critical',
      status: 'firing',
      teamId: 'team-1',
      from: '2026-06-01T00:00:00Z',
      to: '2026-06-30T00:00:00Z',
      labelKey: 'env',
      labelValue: 'prod',
      groupBy: 'label' as const,
      groupLabelKey: 'team',
    };

    const params = filtersToQuery(filters, 2, 50);
    expect(params.get('q')).toBe('cpu');
    expect(params.get('severity')).toBe('critical');
    expect(params.get('status')).toBe('firing');
    expect(params.get('team_id')).toBe('team-1');
    expect(params.get('from')).toBe('2026-06-01T00:00:00Z');
    expect(params.get('to')).toBe('2026-06-30T00:00:00Z');
    expect(params.get('label')).toBe('env:prod');
    expect(params.get('group_by')).toBe('label:team');
    expect(params.get('page')).toBe('2');
    expect(params.get('page_size')).toBe('50');
    expect(params.get('include_analytics')).toBe('true');
    expect(params.get('analytics_label_key')).toBe('team');
  });

  it('serializes severity grouping without label group_by', () => {
    const filters = { ...defaultAlertFilters(), groupBy: 'severity' as const };
    const params = filtersToQuery(filters, 1, 25);
    expect(params.get('group_by')).toBe('severity');
  });

  it('builds export query without pagination and analytics', () => {
    const filters = { ...defaultAlertFilters(), q: 'cpu', groupBy: 'severity' as const };
    const params = filtersToExportQuery(filters);
    expect(params.get('q')).toBe('cpu');
    expect(params.get('page')).toBeNull();
    expect(params.get('include_analytics')).toBeNull();
    expect(params.get('group_by')).toBeNull();
  });

  it('round-trips saved view filters', () => {
    const filters = {
      ...defaultAlertFilters(),
      q: 'disk',
      severity: 'warning',
      status: 'resolved',
      teamId: 'team-2',
      from: '2026-06-01T00:00:00Z',
      to: '2026-06-02T00:00:00Z',
      labelKey: 'region',
      labelValue: 'eu',
      groupBy: 'label' as const,
      groupLabelKey: 'service',
    };

    const saved = filtersToSavedView(filters);
    const restored = savedViewToFilters(saved);

    expect(restored.q).toBe('disk');
    expect(restored.severity).toBe('warning');
    expect(restored.status).toBe('resolved');
    expect(restored.teamId).toBe('team-2');
    expect(restored.from).toBe('2026-06-01T00:00:00Z');
    expect(restored.to).toBe('2026-06-02T00:00:00Z');
    expect(restored.labelKey).toBe('region');
    expect(restored.labelValue).toBe('eu');
    expect(restored.groupBy).toBe('label');
    expect(restored.groupLabelKey).toBe('service');
  });

  it('restores severity grouping from saved view', () => {
    const restored = savedViewToFilters({ group_by: 'severity' });
    expect(restored.groupBy).toBe('severity');
  });

  it('ignores unknown saved view fields', () => {
    const restored = savedViewToFilters({ unknown: true });
    expect(restored).toEqual(defaultAlertFilters());
  });
});
