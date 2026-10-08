import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import { ProtectedRoute } from './components/auth/ProtectedRoute';
import { AppShell, type AppPage } from './components/layout/AppShell';
import { useAuth } from './context/AuthContext';
import { IncidentsPage } from './pages/IncidentsPage';
import { AlertsPage } from './pages/AlertsPage';
import { LoginPage } from './pages/LoginPage';
import { DashboardPage } from './pages/DashboardPage';
import { SettingsPage } from './pages/SettingsPage';
import { BootstrapPage } from './pages/BootstrapPage';
import { AccountPage } from './pages/AccountPage';
import { ShiftsLandingPage } from './pages/ShiftsLandingPage';
import { TeamDetailPage } from './pages/TeamDetailPage';
import { TeamsPage } from './pages/TeamsPage';
import { TeamShiftsRoute } from './pages/TeamShiftsRoute';
import { UsersPage } from './pages/UsersPage';
import { WorkspaceDetailPage } from './pages/WorkspaceDetailPage';
import { WorkspacesPage } from './pages/WorkspacesPage';

function LegacySettingsRedirect({ integrations = false }: { integrations?: boolean }) {
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  if (integrations) { params.set('section', 'integrations'); }
  return <Navigate to={`/settings${params.size ? `?${params}` : ''}`} replace />;
}

function pageFromPath(pathname: string): AppPage {
  if (pathname.includes('/shifts')) {
    return 'shifts';
  }
  if (pathname.startsWith('/workspaces')) {
    return 'workspaces';
  }
  if (pathname.startsWith('/teams')) {
    return 'teams';
  }
  if (pathname.startsWith('/settings') || pathname.startsWith('/setup') || pathname.startsWith('/integrations')) {
    return 'settings';
  }
  if (pathname.startsWith('/dashboard')) {
    return 'dashboard';
  }
  if (pathname.startsWith('/alerts')) {
    return 'alerts';
  }
  if (pathname.startsWith('/incidents')) {
    return 'incidents';
  }
  if (pathname.startsWith('/users')) {
    return 'users';
  }
  return 'shifts';
}

function AppRoutes() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, signOut } = useAuth();
  const currentPage = pageFromPath(location.pathname);

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
          path="/workspaces"
          element={
            <ProtectedRoute>
              <WorkspacesPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/workspaces/:workspaceId"
          element={
            <ProtectedRoute>
              <WorkspaceDetailPage />
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
            <ProtectedRoute>
              <IncidentsPage />
            </ProtectedRoute>
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
              <LegacySettingsRedirect integrations />
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
              <LegacySettingsRedirect />
            </ProtectedRoute>
          }
        />
        <Route path="/settings" element={<ProtectedRoute><SettingsPage /></ProtectedRoute>} />
        <Route
          path="/users"
          element={
            <ProtectedRoute>
              <UsersPage />
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
      <Route path="/bootstrap" element={<BootstrapPage />} />
      <Route path="/*" element={<AppRoutes />} />
    </Routes>
  );
}
