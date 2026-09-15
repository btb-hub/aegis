import { useTranslation } from 'react-i18next';
import type { OnCallUser } from '../../lib/shiftsTypes';
import { PersonContacts } from '../ui/PersonContacts';

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

  return (
    <div className="rounded-lg border border-accent/20 bg-accent/5 px-4 py-3">
      <p className="text-xs font-medium uppercase tracking-wide text-accent">{t('shifts.on_call_now')}</p>
      <ul className="mt-2 space-y-3">
        {users.map((user) => (
          <li key={user.userId}>
            <PersonContacts displayName={user.displayName} contacts={user.contacts} />
          </li>
        ))}
      </ul>
    </div>
  );
}
