import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { UserDirectoryItem } from '../../lib/teamTypes';
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
  const [results, setResults] = useState<UserDirectoryItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const timer = window.setTimeout(() => setDebouncedQuery(query.trim()), 300);
    return () => window.clearTimeout(timer);
  }, [query]);

  const searchUsers = useCallback(async () => {
    if (!debouncedQuery) {
      setResults([]);
      setError(null);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ q: debouncedQuery, page_size: '20' });
      const response = await fetch(`/api/v1/users?${params.toString()}`, { credentials: 'include' });
      if (response.status === 401) {
        setError(t('teams.sign_in_required'));
        setResults([]);
        return;
      }
      if (!response.ok) {
        throw new Error(t('teams.member_picker.load_error'));
      }
      const data = (await response.json()) as { items: UserDirectoryItem[] };
      const excluded = new Set(excludeUserIds);
      setResults((data.items ?? []).filter((user) => !excluded.has(user.id)));
    } catch {
      setError(t('teams.member_picker.load_error'));
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, [debouncedQuery, excludeUserIds, t]);

  useEffect(() => {
    void searchUsers();
  }, [searchUsers]);

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
                  setResults([]);
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
