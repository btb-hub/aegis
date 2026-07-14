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

const teams = [
  { id: 'team-l3-a', name: 'Platform L3' },
  { id: 'team-l3-b', name: 'Data L3' },
];

describe('IncidentDetail', () => {
  it('renders alerts, timeline, jira link, and action buttons', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail
          incident={incident}
          teams={teams}
          owningTeamName="Platform L2"
          owningTier="l2"
          canBounce={false}
          onAcknowledge={vi.fn()}
          onResolve={vi.fn()}
          onHandoff={vi.fn()}
          onBounce={vi.fn()}
        />
      </I18nextProvider>,
    );

    expect(screen.getAllByText('CPU high').length).toBeGreaterThan(0);
    expect(screen.getByRole('link', { name: 'Open OPS-42 in Jira' })).toBeInTheDocument();
    expect(screen.getByText('Acknowledge')).toBeInTheDocument();
    expect(screen.getByText('Resolve')).toBeInTheDocument();
    expect(screen.getByText('Hand off to L3')).toBeInTheDocument();
    expect(screen.getByText('Owned by Platform L2')).toBeInTheDocument();
    expect(screen.getByText('Created')).toBeInTheDocument();
  });

  it('shows actionable messages for skipped integrations', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail
          incident={{
            ...incident,
            timeline: [
              {
                id: 'e2',
                kind: 'integration_skipped',
                payload: {
                  message:
                    'Slack skipped: no global connector. Configure global Slack or set the workspace slot to Custom.',
                },
                createdAt: '2026-06-26T10:01:00Z',
              },
              {
                id: 'e3',
                kind: 'integration_skipped',
                payload: {},
                createdAt: '2026-06-26T10:02:00Z',
              },
            ],
          }}
          teams={teams}
          canBounce={false}
          onAcknowledge={vi.fn()}
          onResolve={vi.fn()}
          onHandoff={vi.fn()}
          onBounce={vi.fn()}
        />
      </I18nextProvider>,
    );

    expect(screen.getByText('Integration skipped')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Slack skipped: no global connector. Configure global Slack or set the workspace slot to Custom.',
      ),
    ).toBeInTheDocument();
  });

  it('calls acknowledge handler', () => {
    const onAcknowledge = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail
          incident={incident}
          teams={teams}
          canBounce={false}
          onAcknowledge={onAcknowledge}
          onResolve={vi.fn()}
          onHandoff={vi.fn()}
          onBounce={vi.fn()}
        />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByText('Acknowledge'));
    expect(onAcknowledge).toHaveBeenCalledWith(incident.id);
  });

  it('calls resolve handler', () => {
    const onResolve = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail
          incident={incident}
          teams={teams}
          canBounce={false}
          onAcknowledge={vi.fn()}
          onResolve={onResolve}
          onHandoff={vi.fn()}
          onBounce={vi.fn()}
        />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByText('Resolve'));
    expect(onResolve).toHaveBeenCalledWith(incident.id);
  });

  it('submits handoff with selected team', () => {
    const onHandoff = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail
          incident={incident}
          teams={teams}
          owningTier="l2"
          canBounce={false}
          onAcknowledge={vi.fn()}
          onResolve={vi.fn()}
          onHandoff={onHandoff}
          onBounce={vi.fn()}
        />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByText('Hand off to L3'));
    fireEvent.change(screen.getByLabelText('L3 team'), { target: { value: 'team-l3-b' } });
    fireEvent.change(screen.getByLabelText('Note'), { target: { value: 'needs L3' } });
    fireEvent.click(screen.getByRole('button', { name: 'Hand off' }));
    expect(onHandoff).toHaveBeenCalledWith(incident.id, 'team-l3-b', 'needs L3');
  });

  it('shows L1 team label when escalating from NOC', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail
          incident={incident}
          teams={[{ id: 'team-l1', name: 'Helpdesk', supportTier: 'l1' }]}
          owningTier="noc"
          canBounce={false}
          onAcknowledge={vi.fn()}
          onResolve={vi.fn()}
          onHandoff={vi.fn()}
          onBounce={vi.fn()}
        />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByText('Escalate to L1'));
    expect(screen.getByLabelText('L1 team')).toBeInTheDocument();
    expect(screen.getByText('Helpdesk (L1)')).toBeInTheDocument();
  });

  it('shows bounce when prior handoff exists', () => {
    const onBounce = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail
          incident={{
            ...incident,
            timeline: [...incident.timeline, { id: 'e2', kind: 'handoff', payload: {}, createdAt: '2026-06-26T11:00:00Z' }],
          }}
          teams={teams}
          canBounce
          onAcknowledge={vi.fn()}
          onResolve={vi.fn()}
          onHandoff={vi.fn()}
          onBounce={onBounce}
        />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByText('Bounce to L2'));
    fireEvent.change(screen.getByLabelText('Reason'), { target: { value: 'wrong team' } });
    fireEvent.click(screen.getByRole('button', { name: 'Bounce' }));
    expect(onBounce).toHaveBeenCalledWith(incident.id, 'wrong team');
  });
});
