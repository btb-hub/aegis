import { fireEvent, render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import { IncidentDetail } from './IncidentDetail';
import i18n from '../../i18n';
import type { Incident } from '../../lib/incidentTypes';

const incident: Incident = {
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
};

describe('IncidentDetail', () => {
  it('renders alerts, timeline, jira link, and action buttons', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail incident={incident} onAcknowledge={vi.fn()} onResolve={vi.fn()} />
      </I18nextProvider>,
    );

    expect(screen.getAllByText('CPU high').length).toBeGreaterThan(0);
    expect(screen.getByRole('link', { name: 'Open OPS-42 in Jira' })).toBeInTheDocument();
    expect(screen.getByText('Acknowledge')).toBeInTheDocument();
    expect(screen.getByText('Resolve')).toBeInTheDocument();
    expect(screen.getByText('Created')).toBeInTheDocument();
  });

  it('calls acknowledge handler', () => {
    const onAcknowledge = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail incident={incident} onAcknowledge={onAcknowledge} onResolve={vi.fn()} />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByText('Acknowledge'));
    expect(onAcknowledge).toHaveBeenCalledWith(incident.id);
  });

  it('calls resolve handler', () => {
    const onResolve = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail incident={incident} onAcknowledge={vi.fn()} onResolve={onResolve} />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByText('Resolve'));
    expect(onResolve).toHaveBeenCalledWith(incident.id);
  });
});
