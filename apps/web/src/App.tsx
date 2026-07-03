import { useMemo, useState } from 'react';
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import { ProtectedRoute } from './components/auth/ProtectedRoute';
import { AppShell, type AppPage } from './components/layout/AppShell';
import { useAuth } from './context/AuthContext';
import type { Incident } from './lib/incidentTypes';
import { IncidentsPage } from './pages/IncidentsPage';
import { AlertsPage } from './pages/AlertsPage';
import { IntegrationsPage } from './pages/IntegrationsPage';
import { LoginPage } from './pages/LoginPage';
import { DashboardPage } from './pages/DashboardPage';
import { SetupWizardPage } from './pages/SetupWizardPage';
import { AccountPage } from './pages/AccountPage';
import { ShiftsLandingPage } from './pages/ShiftsLandingPage';
import { TeamDetailPage } from './pages/TeamDetailPage';
import { TeamsPage } from './pages/TeamsPage';
import { TeamShiftsRoute } from './pages/TeamShiftsRoute';

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
  if (pathname.includes('/shifts')) {
    return 'shifts';
  }
  if (pathname.startsWith('/teams')) {
    return 'teams';
  }
  if (pathname.startsWith('/setup')) {
    return 'setup';
  }
  if (pathname.startsWith('/dashboard')) {
    return 'dashboard';
  }
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

const handoffTeams = [
  { id: 'team-platform-l3', name: 'Platform L3' },
  { id: 'team-data-l3', name: 'Data L3' },
];

function AppRoutes() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, signOut } = useAuth();
  const [incidents, setIncidents] = useState(initialIncidents);

  const currentPage = pageFromPath(location.pathname);

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
      handoff: (incidentId: string, toTeamId: string, note: string) => {
        setIncidents((current) =>
          current.map((incident) =>
            incident.id === incidentId && incident.status !== 'resolved'
              ? {
                  ...incident,
                  timeline: [
                    ...incident.timeline,
                    {
                      id: `event-handoff-${incidentId}`,
                      kind: 'handoff',
                      payload: { to_team_id: toTeamId, note },
                      createdAt: new Date().toISOString(),
                    },
                  ],
                }
              : incident,
          ),
        );
      },
      bounce: (incidentId: string, note: string) => {
        setIncidents((current) =>
          current.map((incident) =>
            incident.id === incidentId && incident.status !== 'resolved'
              ? {
                  ...incident,
                  timeline: [
                    ...incident.timeline,
                    {
                      id: `event-bounce-${incidentId}`,
                      kind: 'bounced',
                      payload: { note },
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
          path="/teams"
          element={
            <ProtectedRoute>
              <TeamsPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/teams/:teamId"
          element={
            <ProtectedRoute>
              <TeamDetailPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/account"
          element={
            <ProtectedRoute>
              <AccountPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/shifts"
          element={
            <ProtectedRoute>
              <ShiftsLandingPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/teams/:teamId/shifts"
          element={
            <ProtectedRoute>
              <TeamShiftsRoute />
            </ProtectedRoute>
          }
        />
        <Route
          path="/incidents"
          element={
            <IncidentsPage
              incidents={incidents}
              handoffTeams={handoffTeams}
              onAcknowledge={handlers.acknowledge}
              onResolve={handlers.resolve}
              onHandoff={handlers.handoff}
              onBounce={handlers.bounce}
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
        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <DashboardPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/setup"
          element={
            <ProtectedRoute>
              <SetupWizardPage />
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
