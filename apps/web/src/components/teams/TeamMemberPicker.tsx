import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiError } from '../../lib/apiClient';
import { queryKeys } from '../../lib/queryClient';
import type { UserDirectoryItem } from '../../lib/teamTypes';
import { fetchUsers } from '../../lib/usersApi';
import { useLoader } from '../../lib/useLoader';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';

type TeamMemberPickerProps = {
  onSelect: (user: UserDirectoryItem) => void;
  excludeUserIds?: string[];
  disabled?: boolean;
};

export function TeamMemberPicker({ onSelect, excludeUserIds = [], disabled = false }: TeamMemberPickerProps) {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');

  useEffect(() => {
    const timer = window.setTimeout(() => setDebouncedQuery(query.trim()), 300);
    return () => window.clearTimeout(timer);
  }, [query]);

  const searchQuery = useLoader(
    queryKeys.users.list(debouncedQuery),
    async () => {
      const data = await fetchUsers(debouncedQuery);
      const excluded = new Set(excludeUserIds);
      return (data.items ?? []).filter((user) => !excluded.has(user.id)) as UserDirectoryItem[];
    },
    { enabled: Boolean(debouncedQuery) },
  );

  const results = debouncedQuery ? (searchQuery.data ?? []) : [];
  const loading = Boolean(debouncedQuery) && searchQuery.loading;
  const error = searchQuery.isError
    ? searchQuery.error instanceof ApiError && searchQuery.error.status === 401
      ? t('teams.sign_in_required')
      : t('teams.member_picker.load_error')
    : null;

  return (
    <div className="space-y-3">
      <Input
        label={t('teams.member_picker.search_label')}
        value={query}
        onChange={setQuery}
        error={error ?? undefined}
      />
      {loading ? <p className="text-sm text-zinc-600">{t('teams.member_picker.loading')}</p> : null}
      {!loading && debouncedQuery && results.length === 0 && !error ? (
        <p className="text-sm text-zinc-600">{t('teams.member_picker.empty')}</p>
      ) : null}
      {results.length > 0 ? (
        <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 bg-white">
          {results.map((user) => (
            <li key={user.id} className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
              <div>
                <p className="font-medium text-zinc-900">{user.display_name || user.email}</p>
                <p className="text-zinc-600">{user.email}</p>
              </div>
              <Button
                variant="secondary"
                disabled={disabled}
                onClick={() => {
                  onSelect(user);
                  setQuery('');
                }}
              >
                {t('teams.member_picker.select')}
              </Button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
