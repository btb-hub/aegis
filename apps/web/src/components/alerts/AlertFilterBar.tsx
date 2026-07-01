import { useTranslation } from 'react-i18next';
import type { AlertFilters } from '../../lib/alertTypes';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';

type AlertFilterBarProps = {
  filters: AlertFilters;
  onChange: (filters: AlertFilters) => void;
  onApply: () => void;
};

const severityOptions = ['', 'critical', 'warning', 'info'];
const statusOptions = ['', 'firing', 'resolved'];

export function AlertFilterBar({ filters, onChange, onApply }: AlertFilterBarProps) {
  const { t } = useTranslation();

  const update = (patch: Partial<AlertFilters>) => {
    onChange({ ...filters, ...patch });
  };

  return (
    <div className="space-y-4 rounded-lg border border-zinc-200 bg-white p-4">
      <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
        <Input
          label={t('alerts.filter.search')}
          value={filters.q}
          onChange={(value) => update({ q: value })}
        />
        <label className="space-y-1 text-sm">
          <span className="font-medium text-zinc-700">{t('alerts.filter.severity')}</span>
          <select
            className="h-9 w-full rounded-md border border-zinc-300 px-3 text-sm"
            value={filters.severity}
            onChange={(event) => update({ severity: event.target.value })}
          >
            <option value="">{t('alerts.filter.any')}</option>
            {severityOptions.filter(Boolean).map((value) => (
              <option key={value} value={value}>
                {t(`incidents.severity.${value}`, { defaultValue: value })}
              </option>
            ))}
          </select>
        </label>
        <label className="space-y-1 text-sm">
          <span className="font-medium text-zinc-700">{t('alerts.filter.status')}</span>
          <select
            className="h-9 w-full rounded-md border border-zinc-300 px-3 text-sm"
            value={filters.status}
            onChange={(event) => update({ status: event.target.value })}
          >
            <option value="">{t('alerts.filter.any')}</option>
            {statusOptions.filter(Boolean).map((value) => (
              <option key={value} value={value}>
                {t(`incidents.alert_status.${value}`, { defaultValue: value })}
              </option>
            ))}
          </select>
        </label>
        <div className="space-y-2">
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
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <label className="space-y-1 text-sm">
          <span className="font-medium text-zinc-700">{t('alerts.filter.group_by')}</span>
          <select
            className="h-9 rounded-md border border-zinc-300 px-3 text-sm"
            value={filters.groupBy}
            onChange={(event) =>
              update({ groupBy: event.target.value as AlertFilters['groupBy'] })
            }
          >
            <option value="">{t('alerts.filter.group_none')}</option>
            <option value="severity">{t('alerts.filter.group_severity')}</option>
            <option value="label">{t('alerts.filter.group_label')}</option>
          </select>
        </label>
        {filters.groupBy === 'label' ? (
          <Input
            label={t('alerts.filter.group_label_key')}
            value={filters.groupLabelKey}
            onChange={(value) => update({ groupLabelKey: value })}
          />
        ) : null}
        <Button onClick={onApply}>{t('alerts.filter.apply')}</Button>
      </div>
    </div>
  );
}
