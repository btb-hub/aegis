import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { AlertAnalyticsPanel } from '../components/alerts/AlertAnalyticsPanel';
import { AlertFilterBar } from '../components/alerts/AlertFilterBar';
import { AlertGroupTable, AlertTable } from '../components/alerts/AlertTable';
import { Banner } from '../components/ui/Banner';
import { Button } from '../components/ui/Button';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Toast } from '../components/ui/Toast';
import { useAuth } from '../context/AuthContext';
import {
  defaultAlertFilters,
  filtersToSavedView,
  savedViewToFilters,
  type AlertFilters,
} from '../lib/alertTypes';
import { createSavedView, exportAlertsCsv, fetchAlerts, fetchSavedViews } from '../lib/alertsApi';
import { ApiError } from '../lib/apiClient';
import { queryKeys } from '../lib/queryClient';
import { useLoader } from '../lib/useLoader';
import { fetchWorkspaces } from '../lib/workspacesApi';

const PAGE_SIZE = 25;

export function AlertsPage() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const navigate = useNavigate();
  const isAdmin = user?.role === 'admin';
  const [draftFilters, setDraftFilters] = useState<AlertFilters>(() => defaultAlertFilters());
  const [appliedFilters, setAppliedFilters] = useState<AlertFilters>(() => defaultAlertFilters());
  const [page, setPage] = useState(1);
  const [selectedViewId, setSelectedViewId] = useState('');
  const [saveName, setSaveName] = useState('');
  const [shareView, setShareView] = useState(false);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);
  const [routingLoading, setRoutingLoading] = useState(false);
  const queryClient = useQueryClient();

  const alertsQuery = useLoader(queryKeys.alerts.list(`${JSON.stringify(appliedFilters)}:${page}`), () =>
    fetchAlerts(appliedFilters, page, PAGE_SIZE),
  );
  const viewsQuery = useLoader(queryKeys.alerts.savedViews, () => fetchSavedViews());
  const alerts = alertsQuery.data;
  const items = alerts?.items ?? [];
  const groups = alerts?.groups ?? [];
  const groupBy = alerts?.group_by ?? '';
  const total = alerts?.total ?? 0;
  const analytics = alerts?.analytics ?? null;
  const savedViews = viewsQuery.data ?? [];
  const loading = alertsQuery.loading;
  const loadError = alertsQuery.isError
    ? alertsQuery.error instanceof ApiError && alertsQuery.error.status === 401
      ? t('alerts.sign_in_required')
      : t('alerts.load_error')
    : null;

  const applyFilters = () => {
    setPage(1);
    setAppliedFilters(draftFilters);
  };

  const loadSavedView = (viewId: string) => {
    setSelectedViewId(viewId);
    const view = savedViews.find((item) => item.id === viewId);
    if (!view) {
      return;
    }
    const next = savedViewToFilters(view.filter);
    setDraftFilters(next);
    setAppliedFilters(next);
    setPage(1);
  };

  const saveCurrentView = async () => {
    if (!saveName.trim()) {
      setToast({ message: t('alerts.saved_views.name_required'), variant: 'default' });
      return;
    }
    try {
      await createSavedView({
        name: saveName.trim(),
        filter: filtersToSavedView(appliedFilters),
        shared: shareView,
      });
      setSaveName('');
      setShareView(false);
      setToast({ message: t('alerts.saved_views.save_success'), variant: 'success' });
      await queryClient.invalidateQueries({ queryKey: queryKeys.alerts.savedViews });
    } catch {
      setToast({ message: t('alerts.saved_views.save_failed'), variant: 'default' });
    }
  };

  const exportCsv = async () => {
    try {
      const blob = await exportAlertsCsv(appliedFilters);
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = 'alerts.csv';
      link.click();
      URL.revokeObjectURL(url);
      setToast({ message: t('alerts.export_success'), variant: 'success' });
    } catch {
      setToast({ message: t('alerts.export_failed'), variant: 'default' });
    }
  };

  const openRouting = async () => {
    setRoutingLoading(true);
    try {
      const workspaces = await fetchWorkspaces();
      if (workspaces.length === 1) {
        navigate(`/workspaces/${workspaces[0].id}`);
      } else {
        navigate('/workspaces');
      }
    } catch {
      setToast({ message: t('alerts.configure_routing_failed'), variant: 'default' });
    } finally {
      setRoutingLoading(false);
    }
  };

  return (
    <PageContent>
      <PageHeader
        title={t('alerts.page_title')}
        subtitle={t('alerts.page_subtitle')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [
            { label: t('nav.platform'), href: '/dashboard' },
            { label: t('nav.alerts') },
          ],
        }}
        actions={
          isAdmin ? (
            <Button variant="secondary" disabled={routingLoading} onClick={() => void openRouting()}>
              {t('alerts.configure_routing')}
            </Button>
          ) : undefined
        }
      />

      <AlertFilterBar
        filters={draftFilters}
        appliedFilters={appliedFilters}
        onChange={setDraftFilters}
        onApply={applyFilters}
        resultTotal={loading ? undefined : total}
        savedViews={savedViews}
        selectedViewId={selectedViewId}
        onLoadView={loadSavedView}
        saveName={saveName}
        onSaveNameChange={setSaveName}
        shareView={shareView}
        onShareViewChange={setShareView}
        onSaveView={() => void saveCurrentView()}
        onExport={() => void exportCsv()}
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}

      {loading ? (
        <p className="text-sm text-zinc-600">{t('alerts.loading')}</p>
      ) : (
        <>
          <AlertAnalyticsPanel analytics={analytics} />
          {groupBy ? (
            <AlertGroupTable groups={groups} groupBy={groupBy} total={total} />
          ) : (
            <AlertTable
              items={items}
              total={total}
              page={page}
              pageSize={PAGE_SIZE}
              onPageChange={setPage}
            />
          )}
        </>
      )}

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
