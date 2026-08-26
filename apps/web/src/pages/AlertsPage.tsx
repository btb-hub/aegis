import { useCallback, useEffect, useState } from 'react';
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
  filtersToExportQuery,
  filtersToQuery,
  filtersToSavedView,
  savedViewToFilters,
  type AlertAnalytics,
  type AlertFilters,
  type AlertGroup,
  type AlertItem,
  type SavedView,
} from '../lib/alertTypes';
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
  const [items, setItems] = useState<AlertItem[]>([]);
  const [groups, setGroups] = useState<AlertGroup[]>([]);
  const [groupBy, setGroupBy] = useState('');
  const [total, setTotal] = useState(0);
  const [analytics, setAnalytics] = useState<AlertAnalytics | null>(null);
  const [savedViews, setSavedViews] = useState<SavedView[]>([]);
  const [selectedViewId, setSelectedViewId] = useState('');
  const [saveName, setSaveName] = useState('');
  const [shareView, setShareView] = useState(false);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);
  const [routingLoading, setRoutingLoading] = useState(false);

  const loadSavedViews = useCallback(async () => {
    const response = await fetch('/api/v1/saved-views', { credentials: 'include' });
    if (!response.ok) {
      return;
    }
    const data = (await response.json()) as { items: SavedView[] };
    setSavedViews(data.items ?? []);
  }, []);

  const loadAlerts = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const params = filtersToQuery(appliedFilters, page, PAGE_SIZE);
      const response = await fetch(`/api/v1/alerts?${params.toString()}`, { credentials: 'include' });
      if (response.status === 401) {
        setLoadError(t('alerts.sign_in_required'));
        return;
      }
      if (!response.ok) {
        throw new Error(t('alerts.load_error'));
      }
      const data = (await response.json()) as {
        items?: AlertItem[];
        groups?: AlertGroup[];
        group_by?: string;
        total?: number;
        analytics?: AlertAnalytics;
      };
      setTotal(data.total ?? 0);
      setAnalytics(data.analytics ?? null);
      if (data.groups) {
        setGroups(data.groups);
        setGroupBy(data.group_by ?? '');
        setItems([]);
      } else {
        setItems(data.items ?? []);
        setGroups([]);
        setGroupBy('');
      }
    } catch {
      setLoadError(t('alerts.load_error'));
      setItems([]);
      setGroups([]);
      setAnalytics(null);
    } finally {
      setLoading(false);
    }
  }, [appliedFilters, page, t]);

  useEffect(() => {
    void loadSavedViews();
  }, [loadSavedViews]);

  useEffect(() => {
    void loadAlerts();
  }, [loadAlerts]);

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
    const response = await fetch('/api/v1/saved-views', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: saveName.trim(),
        filter: filtersToSavedView(appliedFilters),
        shared: shareView,
      }),
    });
    if (!response.ok) {
      setToast({ message: t('alerts.saved_views.save_failed'), variant: 'default' });
      return;
    }
    setSaveName('');
    setShareView(false);
    setToast({ message: t('alerts.saved_views.save_success'), variant: 'success' });
    await loadSavedViews();
  };

  const exportCsv = async () => {
    const params = filtersToExportQuery(appliedFilters);
    const response = await fetch(`/api/v1/alerts/export?${params.toString()}`, {
      credentials: 'include',
    });
    if (!response.ok) {
      setToast({ message: t('alerts.export_failed'), variant: 'default' });
      return;
    }
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'alerts.csv';
    link.click();
    URL.revokeObjectURL(url);
    setToast({ message: t('alerts.export_success'), variant: 'success' });
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
