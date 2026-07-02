import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useNavigate } from 'react-router-dom';
import { MetricTrendChart } from '../components/analytics/MetricTrendChart';
import { Button } from '../components/ui/Button';
import { PageBreadcrumb } from '../components/ui/PageBreadcrumb';
import {
  defaultAnalyticsRange,
  formatDuration,
  type OverviewAnalytics,
} from '../lib/analyticsTypes';

export function DashboardPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [overview, setOverview] = useState<OverviewAnalytics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [comparePrevious, setComparePrevious] = useState(true);

  const loadOverview = useCallback(async () => {
    setLoading(true);
    setError(null);
    const { from, to } = defaultAnalyticsRange();
    const params = new URLSearchParams({
      from,
      to,
      compare_previous: comparePrevious ? 'true' : 'false',
    });
    try {
      const response = await fetch(`/api/v1/analytics/overview?${params.toString()}`, {
        credentials: 'include',
      });
      if (response.status === 401) {
        setError(t('dashboard.sign_in_required'));
        return;
      }
      if (!response.ok) {
        throw new Error(t('dashboard.load_error'));
      }
      setOverview((await response.json()) as OverviewAnalytics);
    } catch {
      setError(t('dashboard.load_error'));
      setOverview(null);
    } finally {
      setLoading(false);
    }
  }, [comparePrevious, t]);

  useEffect(() => {
    void loadOverview();
  }, [loadOverview]);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <PageBreadcrumb
            ariaLabel={t('nav.breadcrumb_label')}
            items={[
              { label: t('shifts.demo_team'), href: '/shifts' },
              { label: t('nav.dashboard') },
            ]}
          />
          <h1 className="text-2xl font-semibold text-zinc-900">{t('dashboard.page_title')}</h1>
          <p className="mt-1 text-sm text-zinc-600">{t('dashboard.page_subtitle')}</p>
        </div>
        <label className="flex items-center gap-2 text-sm text-zinc-700">
          <input
            type="checkbox"
            checked={comparePrevious}
            onChange={(event) => setComparePrevious(event.target.checked)}
            className="h-4 w-4 rounded border-zinc-300 text-accent focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
          />
          {t('dashboard.compare_previous')}
        </label>
      </div>

      {error ? (
        <div role="alert" className="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950">
          {error}
        </div>
      ) : null}

      {loading ? (
        <p className="text-sm text-zinc-600">{t('dashboard.loading')}</p>
      ) : overview ? (
        <div className="grid gap-4 lg:grid-cols-2">
          <section className="rounded-lg border border-zinc-200 bg-white p-4">
            <div className="mb-3 flex items-center justify-between gap-2">
              <h2 className="text-sm font-semibold text-zinc-900">{t('dashboard.mtta_title')}</h2>
              <Button variant="ghost" onClick={() => navigate('/incidents?status=acknowledged')}>
                {t('dashboard.view_incidents')}
              </Button>
            </div>
            <p className="text-2xl font-semibold text-zinc-900">
              {formatDuration(overview.mtta.mean_seconds)}
            </p>
            {overview.mtta.previous ? (
              <p className="text-xs text-zinc-600">
                {t('dashboard.previous_period', {
                  value: formatDuration(overview.mtta.previous.mean_seconds),
                })}
              </p>
            ) : null}
            <MetricTrendChart
              series={overview.mtta.series}
              ariaLabel={t('dashboard.mtta_title')}
              emptyLabel={t('dashboard.empty')}
            />
          </section>

          <section className="rounded-lg border border-zinc-200 bg-white p-4">
            <div className="mb-3 flex items-center justify-between gap-2">
              <h2 className="text-sm font-semibold text-zinc-900">{t('dashboard.mttr_title')}</h2>
              <Button variant="ghost" onClick={() => navigate('/incidents?status=resolved')}>
                {t('dashboard.view_incidents')}
              </Button>
            </div>
            <p className="text-2xl font-semibold text-zinc-900">
              {formatDuration(overview.mttr.mean_seconds)}
            </p>
            {overview.mttr.previous ? (
              <p className="text-xs text-zinc-600">
                {t('dashboard.previous_period', {
                  value: formatDuration(overview.mttr.previous.mean_seconds),
                })}
              </p>
            ) : null}
            <MetricTrendChart
              series={overview.mttr.series}
              ariaLabel={t('dashboard.mttr_title')}
              emptyLabel={t('dashboard.empty')}
            />
          </section>

          <section className="rounded-lg border border-zinc-200 bg-white p-4">
            <div className="mb-3 flex items-center justify-between gap-2">
              <h2 className="text-sm font-semibold text-zinc-900">{t('dashboard.noise_title')}</h2>
              <Button variant="ghost" onClick={() => navigate('/alerts')}>
                {t('dashboard.view_alerts')}
              </Button>
            </div>
            {overview.noise.items.length === 0 ? (
              <p className="text-sm text-zinc-600">{t('dashboard.empty')}</p>
            ) : (
              <ul className="space-y-2 text-sm">
                {overview.noise.items.map((item) => (
                  <li key={item.fingerprint} className="flex items-center justify-between gap-2">
                    <button
                      type="button"
                      className="truncate text-left text-accent hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
                      onClick={() => navigate(`/alerts?q=${encodeURIComponent(item.title)}`)}
                    >
                      {item.title}
                    </button>
                    <span className="shrink-0 text-zinc-600">{item.count}</span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="rounded-lg border border-zinc-200 bg-white p-4">
            <div className="mb-3 flex items-center justify-between gap-2">
              <h2 className="text-sm font-semibold text-zinc-900">{t('dashboard.load_title')}</h2>
              <Button variant="ghost" onClick={() => navigate('/incidents')}>
                {t('dashboard.view_incidents')}
              </Button>
            </div>
            {overview.on_call_load.items.length === 0 ? (
              <p className="text-sm text-zinc-600">{t('dashboard.empty')}</p>
            ) : (
              <ul className="space-y-2 text-sm">
                {overview.on_call_load.items.map((item) => (
                  <li key={item.user_id} className="flex items-center justify-between gap-2">
                    <span className="truncate text-zinc-900">{item.display_name || item.email}</span>
                    <span className="shrink-0 text-zinc-600">
                      {t('dashboard.page_count', { count: item.page_count })}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="rounded-lg border border-zinc-200 bg-white p-4">
            <div className="mb-3 flex items-center justify-between gap-2">
              <h2 className="text-sm font-semibold text-zinc-900">{t('dashboard.handoff_title')}</h2>
              <Button variant="ghost" onClick={() => navigate('/incidents')}>
                {t('dashboard.view_incidents')}
              </Button>
            </div>
            <p className="text-2xl font-semibold text-zinc-900">{overview.handoffs.count}</p>
            <p className="text-sm text-zinc-600">
              {t('dashboard.handoff_median', {
                value: formatDuration(overview.handoffs.median_response_seconds),
              })}
            </p>
          </section>

          <section className="rounded-lg border border-zinc-200 bg-white p-4">
            <div className="mb-3 flex items-center justify-between gap-2">
              <h2 className="text-sm font-semibold text-zinc-900">{t('dashboard.escalation_title')}</h2>
              <Button variant="ghost" onClick={() => navigate('/incidents?status=open')}>
                {t('dashboard.view_incidents')}
              </Button>
            </div>
            <p className="text-2xl font-semibold text-zinc-900">
              {overview.escalation.escalated_percent.toFixed(1)}%
            </p>
            <p className="text-sm text-zinc-600">
              {t('dashboard.escalation_detail', {
                count: overview.escalation.escalated_count,
                total: overview.escalation.total_incidents,
                value: formatDuration(overview.escalation.mean_seconds_to_escalate),
              })}
            </p>
          </section>
        </div>
      ) : null}

      <p className="text-sm text-zinc-600">
        <Link className="text-accent hover:underline" to="/setup">
          {t('dashboard.setup_link')}
        </Link>
      </p>
    </div>
  );
}
