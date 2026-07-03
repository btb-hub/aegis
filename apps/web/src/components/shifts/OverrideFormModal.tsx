import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';
import type { ApiOverride } from '../../lib/shiftsApi';
import type { TeamMember } from '../../lib/teamTypes';

type OverrideFormModalProps = {
  open: boolean;
  onClose: () => void;
  members: TeamMember[];
  overrides: ApiOverride[];
  nameByUserId: Map<string, string>;
  onCreate: (payload: { userId: string; startAt: string; endAt: string }) => Promise<void>;
  onDelete: (overrideId: string) => Promise<void>;
};

function toLocalInputValue(iso: string): string {
  const date = new Date(iso);
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function OverrideFormModal({
  open,
  onClose,
  members,
  overrides,
  nameByUserId,
  onCreate,
  onDelete,
}: OverrideFormModalProps) {
  const { t } = useTranslation();
  const [userId, setUserId] = useState('');
  const [startAt, setStartAt] = useState('');
  const [endAt, setEndAt] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    setUserId(members[0]?.user_id ?? '');
    setStartAt('');
    setEndAt('');
    setError(null);
  }, [open, members]);

  const orderedMembers = useMemo(
    () => [...members].sort((a, b) => a.display_name.localeCompare(b.display_name)),
    [members],
  );

  const handleCreate = async () => {
    if (!userId || !startAt || !endAt) {
      setError(t('override.save_error'));
      return;
    }
    const start = new Date(startAt);
    const end = new Date(endAt);
    if (end <= start) {
      setError(t('override.validation.end_before_start'));
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onCreate({
        userId,
        startAt: start.toISOString(),
        endAt: end.toISOString(),
      });
      setStartAt('');
      setEndAt('');
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : t('override.save_error'));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (overrideId: string) => {
    setDeletingId(overrideId);
    setError(null);
    try {
      await onDelete(overrideId);
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : t('override.delete_error'));
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <Modal
      title={t('override.admin_title')}
      open={open}
      onClose={onClose}
      primaryLabel={t('override.save')}
      secondaryLabel={t('teams.cancel')}
      onPrimary={() => void handleCreate()}
      primaryDisabled={saving}
      primaryLoading={saving}
    >
      <div className="space-y-4">
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-zinc-700">{t('override.user_label')}</span>
          <select
            className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm"
            value={userId}
            onChange={(event) => setUserId(event.target.value)}
          >
            {orderedMembers.map((member) => (
              <option key={member.user_id} value={member.user_id}>
                {member.display_name}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-zinc-700">{t('override.start_label')}</span>
          <input
            type="datetime-local"
            className="h-9 w-full rounded-md border border-zinc-300 px-3 text-sm"
            value={startAt}
            onChange={(event) => setStartAt(event.target.value)}
          />
        </label>
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-zinc-700">{t('override.end_label')}</span>
          <input
            type="datetime-local"
            className="h-9 w-full rounded-md border border-zinc-300 px-3 text-sm"
            value={endAt}
            onChange={(event) => setEndAt(event.target.value)}
          />
        </label>

        {overrides.length > 0 ? (
          <ul className="divide-y divide-zinc-200 rounded-md border border-zinc-200 text-sm">
            {overrides.map((override) => (
              <li key={override.id} className="flex items-center justify-between px-3 py-2">
                <span>
                  {nameByUserId.get(override.user_id) ?? override.user_id}
                  {' · '}
                  {toLocalInputValue(override.start_at)} – {toLocalInputValue(override.end_at)}
                </span>
                <Button
                  variant="ghost"
                  disabled={deletingId === override.id}
                  onClick={() => void handleDelete(override.id)}
                >
                  {t('teams.delete')}
                </Button>
              </li>
            ))}
          </ul>
        ) : null}

        {error ? <p className="text-sm text-red-700">{error}</p> : null}
      </div>
    </Modal>
  );
}
