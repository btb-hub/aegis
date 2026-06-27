import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AppShell } from './components/layout/AppShell';
import type { AppPage } from './components/layout/AppShell';
import type { Incident } from './lib/incidentTypes';
import { IncidentsPage } from './pages/IncidentsPage';
import { TeamShiftsPage } from './pages/TeamShiftsPage';

const demoSlots = [
  {
    id: 'slot-1',
    userId: 'user-alice',
    displayName: 'Alice',
    startAt: '2026-06-02T09:00:00Z',
    endAt: '2026-06-09T09:00:00Z',
    source: 'rotation' as const,
  },
  {
    id: 'slot-2',
    userId: 'user-bob',
    displayName: 'Bob',
    startAt: '2026-06-09T09:00:00Z',
    endAt: '2026-06-16T09:00:00Z',
    source: 'rotation' as const,
  },
];

const demoOverrides = [
  {
    id: 'override-1',
    userId: 'user-carol',
    displayName: 'Carol',
    startAt: '2026-06-12T00:00:00Z',
    endAt: '2026-06-13T00:00:00Z',
  },
];

const initialIncidents: Incident[] = [
  {
    id: '11111111-1111-1111-1111-111111111111',
    teamId: 'team-platform',
    status: 'open',
    severity: 'critical',
    title: 'CPU high on api-1',
    fingerprint: 'cpu-api-1',
    jiraIssueKey: 'OPS-42',
    createdAt: '2026-06-26T10:00:00Z',
    alerts: [{ id: 'alert-1', severity: 'critical', title: 'CPU high on api-1', status: 'firing' }],
    timeline: [
      { id: 'event-1', kind: 'created', payload: {}, createdAt: '2026-06-26T10:00:00Z' },
      { id: 'event-2', kind: 'paged', payload: { channel: 'slack' }, createdAt: '2026-06-26T10:01:00Z' },
    ],
  },
  {
    id: '22222222-2222-2222-2222-222222222222',
    teamId: 'team-platform',
    status: 'acknowledged',
    severity: 'warning',
    title: 'Elevated error rate',
    fingerprint: 'errors-api',
    jiraIssueKey: 'OPS-41',
    createdAt: '2026-06-26T08:00:00Z',
    acknowledgedAt: '2026-06-26T08:05:00Z',
    alerts: [{ id: 'alert-2', severity: 'warning', title: 'Elevated error rate', status: 'firing' }],
    timeline: [
      { id: 'event-3', kind: 'created', payload: {}, createdAt: '2026-06-26T08:00:00Z' },
      { id: 'event-4', kind: 'acknowledged', payload: {}, createdAt: '2026-06-26T08:05:00Z' },
    ],
  },
];

export function App() {
  const { t } = useTranslation();
  const [page, setPage] = useState<AppPage>('shifts');
  const [incidents, setIncidents] = useState(initialIncidents);

  const handlers = useMemo(
    () => ({
      acknowledge: (incidentId: string) => {
        setIncidents((current) =>
          current.map((incident) =>
            incident.id === incidentId && incident.status === 'open'
              ? {
                  ...incident,
                  status: 'acknowledged',
                  acknowledgedAt: new Date().toISOString(),
                  timeline: [
                    ...incident.timeline,
                    {
                      id: `event-ack-${incidentId}`,
                      kind: 'acknowledged',
                      payload: {},
                      createdAt: new Date().toISOString(),
                    },
                  ],
                }
              : incident,
          ),
        );
      },
      resolve: (incidentId: string) => {
        setIncidents((current) =>
          current.map((incident) =>
            incident.id === incidentId && incident.status !== 'resolved'
              ? {
                  ...incident,
                  status: 'resolved',
                  resolvedAt: new Date().toISOString(),
                  timeline: [
                    ...incident.timeline,
                    {
                      id: `event-resolve-${incidentId}`,
                      kind: 'resolved',
                      payload: {},
                      createdAt: new Date().toISOString(),
                    },
                  ],
                }
              : incident,
          ),
        );
      },
    }),
    [],
  );

  return (
    <AppShell currentPage={page} onNavigate={setPage}>
      {page === 'shifts' ? (
        <TeamShiftsPage
          teamName={t('shifts.demo_team')}
          onCallUsers={[{ userId: 'user-bob', displayName: 'Bob', email: 'bob@example.com', source: 'rotation' }]}
          slots={demoSlots}
          overrides={demoOverrides}
          month={new Date('2026-06-01T00:00:00Z')}
        />
      ) : (
        <IncidentsPage
          incidents={incidents}
          onAcknowledge={handlers.acknowledge}
          onResolve={handlers.resolve}
        />
      )}
    </AppShell>
  );
}
