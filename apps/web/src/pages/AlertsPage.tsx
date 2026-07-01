import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AlertAnalyticsPanel } from '../components/alerts/AlertAnalyticsPanel';
import { AlertFilterBar } from '../components/alerts/AlertFilterBar';
import { AlertGroupTable, AlertTable } from '../components/alerts/AlertTable';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { PageBreadcrumb } from '../components/ui/PageBreadcrumb';
import { Toast } from '../components/ui/Toast';
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

const PAGE_SIZE = 25;

export function AlertsPage() {
  const { t } = useTranslation();
  const [draftFilters, setDraftFilters] = useState<AlertFilters>(defaultAlertFilters);
  const [appliedFilters, setAppliedFilters] = useState<AlertFilters>(defaultAlertFilters);
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

  return (
    <div className="space-y-6">
      <div>
        <PageBreadcrumb
          ariaLabel={t('nav.breadcrumb_label')}
          items={[
            { label: t('shifts.demo_team'), href: '/shifts' },
            { label: t('nav.alerts') },
          ]}
        />
        <div className="mt-2 flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold text-zinc-900">{t('alerts.page_title')}</h1>
            <p className="mt-1 text-sm text-zinc-600">{t('alerts.page_subtitle')}</p>
          </div>
          <Button variant="secondary" onClick={() => void exportCsv()}>
            {t('alerts.export')}
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap items-end gap-3 rounded-lg border border-zinc-200 bg-white p-4">
        <label className="space-y-1 text-sm">
          <span className="text-zinc-600">{t('alerts.saved_views.load')}</span>
          <select
            className="h-9 min-w-48 rounded-md border border-zinc-200 px-3 text-sm"
            value={selectedViewId}
            onChange={(event) => loadSavedView(event.target.value)}
          >
            <option value="">{t('alerts.saved_views.none')}</option>
            {savedViews.map((view) => (
              <option key={view.id} value={view.id}>
                {view.name}
                {view.shared ? ` (${t('alerts.saved_views.shared')})` : ''}
              </option>
            ))}
          </select>
        </label>
        <Input
          label={t('alerts.saved_views.name')}
          value={saveName}
          onChange={setSaveName}
        />
        <label className="flex items-center gap-2 text-sm text-zinc-700">
          <input
            type="checkbox"
            checked={shareView}
            onChange={(event) => setShareView(event.target.checked)}
          />
          {t('alerts.saved_views.share')}
        </label>
        <Button variant="secondary" onClick={() => void saveCurrentView()}>
          {t('alerts.saved_views.save')}
        </Button>
      </div>

      <AlertFilterBar filters={draftFilters} onChange={setDraftFilters} onApply={applyFilters} />

      {loadError ? (
        <div
          role="alert"
          className="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950"
        >
          {loadError}
        </div>
      ) : null}

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
    </div>
  );
}
