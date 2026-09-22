import { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import type { Incident } from '../../lib/incidentTypes';
import { AuthProvider } from '../../context/AuthContext';
import { AppShell } from '../layout/AppShell';
import { PageContent } from '../ui/PageContent';
import { PageHeader } from '../ui/PageHeader';
import { IncidentDetail } from './IncidentDetail';
import { IncidentList } from './IncidentList';

const longTitle =
  'HelmRelease is not in the Ready state. HelmRelease amateur-teams-uniform in flux-system is not Ready for more than 20 minutes. Where: alertname: FluxHelmReleaseNotReady namespace: flux-system severity: critical priority: P2 service: kps-kube-state-metrics generatorURL: /graph?g0.expr=gotk_resource_info%7Bcustomresource_group%3D%22helm.toolkit.fluxcd.io%22%7D';

const incidents: Incident[] = [
  {
    id: '11111111-1111-1111-1111-111111111111',
    teamId: 'team-1',
    status: 'acknowledged',
    severity: 'critical',
    title: longTitle,
    fingerprint: 'fp-1',
    createdAt: '2026-09-22T07:57:15.269Z',
    assignee: {
      userId: 'user-1',
      displayName: 'Alex Morgan',
      email: 'alex@example.com',
      contacts: { email: 'mailto:alex@example.com' },
    },
    alerts: [
      { id: 'alert-1', severity: 'critical', title: longTitle, status: 'firing' },
      { id: 'alert-2', severity: 'warning', title: 'Deployment rollout is progressing slowly', status: 'firing' },
    ],
    timeline: [
      { id: 'event-1', kind: 'created', payload: {}, createdAt: '2026-09-22T07:57:15.269Z' },
      { id: 'event-2', kind: 'acknowledged', payload: {}, createdAt: '2026-09-22T08:02:15.269Z' },
    ],
  },
  {
    id: '22222222-2222-2222-2222-222222222222',
    teamId: 'team-1',
    status: 'open',
    severity: 'critical',
    title: 'GitLab health check is failing from the production blackbox exporter',
    fingerprint: 'fp-2',
    createdAt: '2026-09-22T08:17:15.269Z',
    alerts: [],
    timeline: [],
  },
  {
    id: '33333333-3333-3333-3333-333333333333',
    teamId: 'team-1',
    status: 'resolved',
    severity: 'warning',
    title: 'Load balancer probe recovered',
    fingerprint: 'fp-3',
    createdAt: '2026-09-22T06:17:15.269Z',
    alerts: [],
    timeline: [],
  },
];

function IncidentExperience() {
  const [selectedId, setSelectedId] = useState(incidents[0].id);
  const [status, setStatus] = useState<'all' | Incident['status']>('all');
  const selected = incidents.find((incident) => incident.id === selectedId) ?? incidents[0];

  return (
    <MemoryRouter>
      <AuthProvider>
        <AppShell
          currentPage="incidents"
          onNavigate={() => undefined}
          user={{
            id: 'user-1',
            email: 'alex@example.com',
            display_name: 'Alex Morgan',
            role: 'admin',
            locale: 'en',
            provider: 'google',
          }}
        >
          <PageContent>
            <PageHeader title="Incidents" subtitle="Track open incidents, linked alerts, and timeline events" />
            <div className="grid min-w-0 gap-6 xl:grid-cols-[minmax(20rem,0.8fr)_minmax(0,1.35fr)]">
              <IncidentList
                incidents={incidents}
                statusFilter={status}
                onStatusFilterChange={setStatus}
                onSelect={setSelectedId}
                selectedId={selectedId}
              />
              <IncidentDetail
                incident={selected}
                teams={[]}
                owningTeamName="DevOps"
                owningTier="l2"
                canBounce={false}
                onAcknowledge={() => undefined}
                onResolve={() => undefined}
                onHandoff={() => undefined}
                onBounce={() => undefined}
              />
            </div>
          </PageContent>
        </AppShell>
      </AuthProvider>
    </MemoryRouter>
  );
}

const meta: Meta<typeof IncidentExperience> = {
  title: 'Incidents/IncidentExperience',
  component: IncidentExperience,
  parameters: { layout: 'fullscreen' },
};

export default meta;
type Story = StoryObj<typeof IncidentExperience>;

export const English: Story = {};

export const Russian: Story = {
  globals: { locale: 'ru' },
};
