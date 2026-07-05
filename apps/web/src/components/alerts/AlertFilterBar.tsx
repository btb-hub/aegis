import { useTranslation } from 'react-i18next';
import type { AlertFilters, SavedView } from '../../lib/alertTypes';
import { Button } from '../ui/Button';
import { Checkbox } from '../ui/Checkbox';
import { Input } from '../ui/Input';
import { Select } from '../ui/Select';

type AlertFilterBarProps = {
  filters: AlertFilters;
  appliedFilters: AlertFilters;
  onChange: (filters: AlertFilters) => void;
  onApply: () => void;
  resultTotal?: number;
  savedViews: SavedView[];
  selectedViewId: string;
  onLoadView: (viewId: string) => void;
  saveName: string;
  onSaveNameChange: (value: string) => void;
  shareView: boolean;
  onShareViewChange: (value: boolean) => void;
  onSaveView: () => void;
  onExport: () => void;
};

const severityOptions = ['', 'critical', 'warning', 'info'];
const statusOptions = ['', 'firing', 'resolved'];

function activeFilterChips(filters: AlertFilters, t: (key: string, opts?: Record<string, string>) => string) {
  const chips: string[] = [];
  if (filters.q) {
    chips.push(t('alerts.filter.chip_search', { value: filters.q }));
  }
  if (filters.severity) {
    chips.push(t('alerts.filter.chip_severity', { value: filters.severity }));
  }
  if (filters.status) {
    chips.push(t('alerts.filter.chip_status', { value: filters.status }));
  }
  if (filters.labelKey) {
    const label = filters.labelValue
      ? `${filters.labelKey}:${filters.labelValue}`
      : filters.labelKey;
    chips.push(t('alerts.filter.chip_label', { value: label }));
  }
  if (filters.groupBy) {
    const groupValue =
      filters.groupBy === 'label' && filters.groupLabelKey
        ? `${filters.groupBy}:${filters.groupLabelKey}`
        : filters.groupBy;
    chips.push(t('alerts.filter.chip_group', { value: groupValue }));
  }
  return chips;
}

export function AlertFilterBar({
  filters,
  appliedFilters,
  onChange,
  onApply,
  resultTotal,
  savedViews,
  selectedViewId,
  onLoadView,
  saveName,
  onSaveNameChange,
  shareView,
  onShareViewChange,
  onSaveView,
  onExport,
}: AlertFilterBarProps) {
  const { t } = useTranslation();

  const update = (patch: Partial<AlertFilters>) => {
    onChange({ ...filters, ...patch });
  };

  const savedViewOptions = [
    { value: '', label: t('alerts.saved_views.none') },
    ...savedViews.map((view) => ({
      value: view.id,
      label: `${view.name}${view.shared ? ` (${t('alerts.saved_views.shared')})` : ''}`,
    })),
  ];

  const chips = activeFilterChips(appliedFilters, t);

  return (
    <section className="space-y-4 rounded-lg border border-zinc-200 bg-white p-4 shadow-sm">
      <div className="flex flex-wrap items-end gap-3">
        <div className="min-w-[200px] flex-1">
          <Input
            label={t('alerts.filter.search')}
            value={filters.q}
            onChange={(value) => update({ q: value })}
          />
        </div>
        <div className="w-full sm:w-48">
          <Select
            label={t('alerts.saved_views.load')}
            value={selectedViewId}
            options={savedViewOptions}
            onChange={onLoadView}
          />
        </div>
        <Button variant="secondary" onClick={onExport}>
          {t('alerts.export')}
        </Button>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Select
          label={t('alerts.filter.severity')}
          value={filters.severity}
          options={[
            { value: '', label: t('alerts.filter.any') },
            ...severityOptions.filter(Boolean).map((value) => ({
              value,
              label: t(`incidents.severity.${value}`, { defaultValue: value }),
            })),
          ]}
          onChange={(value) => update({ severity: value })}
        />
        <Select
          label={t('alerts.filter.status')}
          value={filters.status}
          options={[
            { value: '', label: t('alerts.filter.any') },
            ...statusOptions.filter(Boolean).map((value) => ({
              value,
              label: t(`incidents.alert_status.${value}`, { defaultValue: value }),
            })),
          ]}
          onChange={(value) => update({ status: value })}
        />
        <Input
          label={t('alerts.filter.label_key')}
          value={filters.labelKey}
          onChange={(value) => update({ labelKey: value })}
        />
        <Input
          label={t('alerts.filter.label_value')}
          value={filters.labelValue}
          onChange={(value) => update({ labelValue: value })}
        />
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Select
          label={t('alerts.filter.group_by')}
          value={filters.groupBy}
          options={[
            { value: '', label: t('alerts.filter.group_none') },
            { value: 'severity', label: t('alerts.filter.group_severity') },
            { value: 'label', label: t('alerts.filter.group_label') },
          ]}
          onChange={(value) => update({ groupBy: value as AlertFilters['groupBy'] })}
        />
        {filters.groupBy === 'label' ? (
          <Input
            label={t('alerts.filter.group_label_key')}
            value={filters.groupLabelKey}
            onChange={(value) => update({ groupLabelKey: value })}
          />
        ) : (
          <div aria-hidden="true" className="hidden lg:block" />
        )}
        <Input
          label={t('alerts.saved_views.name')}
          value={saveName}
          onChange={onSaveNameChange}
        />
        <div className="flex flex-col justify-end gap-3">
          <Checkbox
            label={t('alerts.saved_views.share')}
            checked={shareView}
            onChange={onShareViewChange}
          />
          <Button variant="secondary" onClick={onSaveView}>
            {t('alerts.saved_views.save')}
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-zinc-100 pt-4">
        <div className="flex min-h-9 flex-wrap items-center gap-2 text-sm text-zinc-600">
          {chips.length > 0 ? (
            chips.map((chip) => (
              <span
                key={chip}
                className="rounded-full border border-zinc-200 bg-zinc-50 px-2.5 py-1 font-mono text-xs text-zinc-700"
              >
                {chip}
              </span>
            ))
          ) : (
            <span>{t('alerts.filter.no_active_filters')}</span>
          )}
          {typeof resultTotal === 'number' ? (
            <span className="text-zinc-500">{t('alerts.filter.results', { count: resultTotal })}</span>
          ) : null}
        </div>
        <Button onClick={onApply}>{t('alerts.filter.apply')}</Button>
      </div>
    </section>
  );
}
