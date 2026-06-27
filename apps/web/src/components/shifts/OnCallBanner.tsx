import { useTranslation } from 'react-i18next';
import type { OnCallUser } from '../../lib/shiftsTypes';

type OnCallBannerProps = {
  users: OnCallUser[];
};

export function OnCallBanner({ users }: OnCallBannerProps) {
  const { t } = useTranslation();

  if (users.length === 0) {
    return (
      <div className="rounded-lg border border-zinc-200 bg-surface px-4 py-3 text-sm text-zinc-600">
        {t('shifts.on_call_empty')}
      </div>
    );
  }

  const names = users.map((user) => user.displayName).join(', ');

  return (
    <div className="rounded-lg border border-accent/20 bg-accent/5 px-4 py-3">
      <p className="text-xs font-medium uppercase tracking-wide text-accent">{t('shifts.on_call_now')}</p>
      <p className="mt-1 text-lg font-semibold text-zinc-900">{names}</p>
    </div>
  );
}
