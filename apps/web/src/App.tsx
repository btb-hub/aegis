import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import { ProtectedRoute } from './components/auth/ProtectedRoute';
import { AppShell, type AppPage } from './components/layout/AppShell';
import { useAuth } from './context/AuthContext';
import type { Incident } from './lib/incidentTypes';
import { buildDemoShiftsForMonth, resolveCurrentOnCall } from './lib/shiftsDemoData';
import { IncidentsPage } from './pages/IncidentsPage';
import { AlertsPage } from './pages/AlertsPage';
import { IntegrationsPage } from './pages/IntegrationsPage';
import { LoginPage } from './pages/LoginPage';
import { TeamShiftsPage } from './pages/TeamShiftsPage';

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

function pageFromPath(pathname: string): AppPage {
  if (pathname.startsWith('/integrations')) {
    return 'integrations';
  }
  if (pathname.startsWith('/alerts')) {
    return 'alerts';
  }
  if (pathname.startsWith('/incidents')) {
    return 'incidents';
  }
  return 'shifts';
}

function AppRoutes() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const { user, signOut } = useAuth();
  const [incidents, setIncidents] = useState(initialIncidents);

  const currentPage = pageFromPath(location.pathname);

  const shiftsMonth = useMemo(() => new Date(), []);
  const { slots: demoSlots, overrides: demoOverrides } = useMemo(
    () => buildDemoShiftsForMonth(shiftsMonth),
    [shiftsMonth],
  );
  const onCallUsers = useMemo(
    () => resolveCurrentOnCall(demoSlots, demoOverrides, shiftsMonth),
    [demoSlots, demoOverrides, shiftsMonth],
  );

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
    <AppShell
      currentPage={currentPage}
      onNavigate={(page) => navigate(`/${page}`)}
      user={user}
      onSignOut={signOut}
    >
      <Routes>
        <Route path="/" element={<Navigate to="/shifts" replace />} />
        <Route
          path="/shifts"
          element={
            <TeamShiftsPage
              teamName={t('shifts.demo_team')}
              onCallUsers={onCallUsers}
              slots={demoSlots}
              overrides={demoOverrides}
              month={shiftsMonth}
            />
          }
        />
        <Route
          path="/incidents"
          element={
            <IncidentsPage
              incidents={incidents}
              onAcknowledge={handlers.acknowledge}
              onResolve={handlers.resolve}
            />
          }
        />
        <Route
          path="/alerts"
          element={
            <ProtectedRoute>
              <AlertsPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/integrations"
          element={
            <ProtectedRoute>
              <IntegrationsPage />
            </ProtectedRoute>
          }
        />
        <Route path="*" element={<Navigate to="/shifts" replace />} />
      </Routes>
    </AppShell>
  );
}

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/*" element={<AppRoutes />} />
    </Routes>
  );
}
