import { fireEvent, render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import { IncidentList } from './IncidentList';
import i18n from '../../i18n';
import type { Incident } from '../../lib/incidentTypes';

const incidents: Incident[] = [
  {
    id: '11111111-1111-1111-1111-111111111111',
    teamId: 'team-1',
    status: 'open',
    severity: 'critical',
    title: 'CPU high',
    fingerprint: 'fp-1',
    jiraIssueKey: 'OPS-42',
    createdAt: '2026-06-26T10:00:00Z',
    alerts: [{ id: 'a1', severity: 'critical', title: 'CPU high', status: 'firing' }],
    timeline: [{ id: 'e1', kind: 'created', payload: {}, createdAt: '2026-06-26T10:00:00Z' }],
  },
  {
    id: '22222222-2222-2222-2222-222222222222',
    teamId: 'team-1',
    status: 'resolved',
    severity: 'warning',
    title: 'Disk low',
    fingerprint: 'fp-2',
    createdAt: '2026-06-25T10:00:00Z',
    resolvedAt: '2026-06-25T12:00:00Z',
    alerts: [],
    timeline: [{ id: 'e2', kind: 'resolved', payload: {}, createdAt: '2026-06-25T12:00:00Z' }],
  },
];

describe('IncidentList', () => {
  it('filters incidents by status', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentList
          incidents={incidents}
          statusFilter="open"
          onStatusFilterChange={() => undefined}
          onSelect={() => undefined}
        />
      </I18nextProvider>,
    );

    expect(screen.getByText('CPU high')).toBeInTheDocument();
    expect(screen.queryByText('Disk low')).not.toBeInTheDocument();
  });

  it('calls onSelect when an incident is clicked', () => {
    const onSelect = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentList
          incidents={incidents}
          statusFilter="all"
          onStatusFilterChange={() => undefined}
          onSelect={onSelect}
        />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByText('CPU high'));
    expect(onSelect).toHaveBeenCalledWith('11111111-1111-1111-1111-111111111111');
  });

  it('calls onStatusFilterChange when a filter is clicked', () => {
    const onStatusFilterChange = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentList
          incidents={incidents}
          statusFilter="all"
          onStatusFilterChange={onStatusFilterChange}
          onSelect={() => undefined}
        />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Open' }));
    expect(onStatusFilterChange).toHaveBeenCalledWith('open');
  });
});
