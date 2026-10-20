import { fireEvent, render, screen, waitFor } from '@testing-library/react';
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
    expect(screen.getByText('No assignee')).toBeInTheDocument();
  });

  it('shows assignee contact links', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail
          incident={{
            ...incident,
            assignee: {
              userId: 'user-1',
              email: 'alice@example.com',
              displayName: 'Alice',
              contacts: {
                email: 'mailto:alice@example.com',
                slack: 'https://slack.com/app_redirect?channel=U123',
              },
            },
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

    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Email' })).toHaveAttribute('href', 'mailto:alice@example.com');
    expect(screen.getByRole('link', { name: 'Message in Slack' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Message in eXpress' })).not.toBeInTheDocument();
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

  it('separates the incident summary from raw alert context', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <IncidentDetail
          incident={{
            ...incident,
            title: 'HelmRelease is not ready > 10 min. Where: alertname: FluxHelmReleaseNotReady namespace: flux-system',
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

    expect(screen.getByRole('heading', { name: 'HelmRelease is not ready > 10 min.' })).toBeInTheDocument();
    expect(screen.getByText('Alert context')).toBeInTheDocument();
    expect(screen.getByText(/FluxHelmReleaseNotReady/)).toBeInTheDocument();
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
  it('keeps handoff notes after failure and prevents duplicate submissions while pending', async () => {
    let finish!: (result: boolean) => void;
    const onHandoff = vi.fn(() => new Promise<boolean>((resolve) => { finish = resolve; }));
    render(<I18nextProvider i18n={i18n}><IncidentDetail incident={incident} teams={teams} owningTier="l2" canBounce={false} onAcknowledge={vi.fn()} onResolve={vi.fn()} onHandoff={onHandoff} onBounce={vi.fn()} /></I18nextProvider>);
    fireEvent.click(screen.getByText('Hand off to L3'));
    const note = screen.getByLabelText(i18n.t('incidents.handoff_note_label'));
    fireEvent.change(note, { target: { value: 'Investigate database replication' } });
    fireEvent.click(screen.getByRole('button', { name: 'Hand off' }));
    expect(screen.getByRole('button', { name: 'Hand off' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: 'Hand off' }));
    expect(onHandoff).toHaveBeenCalledTimes(1);
    finish(false);
    await waitFor(() => expect(screen.getByRole('button', { name: 'Hand off' })).not.toBeDisabled());
    expect(note).toHaveValue('Investigate database replication');
    onHandoff.mockResolvedValueOnce(true);
    fireEvent.click(screen.getByRole('button', { name: 'Hand off' }));
    await waitFor(() => expect(screen.queryByLabelText(i18n.t('incidents.handoff_note_label'))).not.toBeInTheDocument());
    expect(screen.getByRole('status')).toHaveTextContent(i18n.t('incidents.action_complete'));
  });

  it('shows a failed action and retains the bounce note', async () => {
    render(<I18nextProvider i18n={i18n}><IncidentDetail incident={incident} teams={teams} owningTier="l3" canBounce onAcknowledge={vi.fn()} onResolve={vi.fn()} onHandoff={vi.fn()} onBounce={vi.fn().mockRejectedValue(new Error('offline'))} /></I18nextProvider>);
    fireEvent.click(screen.getByText('Bounce to L2'));
    fireEvent.change(screen.getByLabelText(i18n.t('incidents.bounce_note_label')), { target: { value: 'Need more logs' } });
    fireEvent.click(screen.getByRole('button', { name: 'Bounce' }));
    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent(i18n.t('incidents.action_error')));
    expect(screen.getByLabelText(i18n.t('incidents.bounce_note_label'))).toHaveValue('Need more logs');
  });

  it('preserves the selected handoff destination through parent error rerenders', () => {
    const renderDetail = (options: typeof teams) => <I18nextProvider i18n={i18n}><IncidentDetail incident={incident} teams={options} owningTier="l2" canBounce={false} onAcknowledge={vi.fn()} onResolve={vi.fn()} onHandoff={vi.fn()} onBounce={vi.fn()} /></I18nextProvider>;
    const { rerender } = render(renderDetail(teams));
    fireEvent.click(screen.getByText('Hand off to L3'));
    const select = screen.getByRole('combobox');
    fireEvent.change(select, { target: { value: 'team-l3-b' } });
    rerender(renderDetail([...teams]));
    expect(select).toHaveValue('team-l3-b');
    rerender(renderDetail([teams[0]]));
    expect(select).toHaveValue('team-l3-a');
  });

  it('removes handoff and bounce forms when the incident is resolved', () => {
    const renderDetail = (status: Incident['status']) => <I18nextProvider i18n={i18n}><IncidentDetail incident={{ ...incident, status }} teams={teams} owningTier="l2" canBounce onAcknowledge={vi.fn()} onResolve={vi.fn()} onHandoff={vi.fn()} onBounce={vi.fn()} /></I18nextProvider>;
    const { rerender } = render(renderDetail('open'));
    fireEvent.click(screen.getByText('Hand off to L3'));
    fireEvent.click(screen.getByText('Bounce to L2'));
    expect(screen.getAllByRole('textbox')).toHaveLength(2);
    rerender(renderDetail('resolved'));
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Hand off' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Bounce' })).not.toBeInTheDocument();
  });

  it('returns focus to the incident heading after a successful form submission', async () => {
    render(<I18nextProvider i18n={i18n}><IncidentDetail incident={incident} teams={teams} owningTier="l2" canBounce={false} onAcknowledge={vi.fn()} onResolve={vi.fn()} onHandoff={vi.fn().mockResolvedValue(true)} onBounce={vi.fn()} /></I18nextProvider>);
    fireEvent.click(screen.getByText('Hand off to L3'));
    const submit = screen.getByRole('button', { name: 'Hand off' });
    submit.focus();
    fireEvent.click(submit);
    await waitFor(() => expect(screen.getByRole('heading', { name: 'CPU high' })).toHaveFocus());
  });

});
