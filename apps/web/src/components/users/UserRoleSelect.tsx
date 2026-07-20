import { useTranslation } from 'react-i18next';
import { Select } from '../ui/Select';
import type { UserRole } from '../../lib/usersApi';

const ROLES: UserRole[] = ['admin', 'member', 'viewer'];

type UserRoleSelectProps = {
  id?: string;
  label: string;
  hideLabel?: boolean;
  value: UserRole;
  pinned?: boolean;
  disabled?: boolean;
  onChange: (role: UserRole) => void;
};

export function UserRoleSelect({
  id,
  label,
  hideLabel = false,
  value,
  pinned = false,
  disabled = false,
  onChange,
}: UserRoleSelectProps) {
  const { t } = useTranslation();

  return (
    <div className="space-y-1">
      <Select
        id={id}
        label={label}
        hideLabel={hideLabel}
        value={value}
        disabled={disabled || pinned}
        options={ROLES.map((role) => ({ value: role, label: t(`users.role.${role}`) }))}
        onChange={(next) => onChange(next as UserRole)}
      />
      {pinned ? <p className="text-xs text-zinc-500">{t('users.pinned')}</p> : null}
    </div>
  );
}
