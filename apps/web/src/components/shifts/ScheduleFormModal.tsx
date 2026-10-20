import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Select } from '../ui/Select';
import { Checkbox } from '../ui/Checkbox';
import { Modal } from '../ui/Modal';
import type { ApiSchedule } from '../../lib/shiftsApi';
import type { TeamMember } from '../../lib/teamTypes';

type ScheduleFormModalProps = {
  open: boolean;
  onClose: () => void;
  members: TeamMember[];
  schedule?: ApiSchedule | null;
  onSave: (payload: {
    name: string;
    timezone: string;
    handoffWeekday: number;
    handoffTime: string;
    participants: string[];
  }) => Promise<void>;
};

const WEEKDAYS = [0, 1, 2, 3, 4, 5, 6];

export function ScheduleFormModal({ open, onClose, members, schedule, onSave }: ScheduleFormModalProps) {
  const { t } = useTranslation();
  const [name, setName] = useState('Primary');
  const [timezone, setTimezone] = useState('UTC');
  const [handoffWeekday, setHandoffWeekday] = useState(1);
  const [handoffTime, setHandoffTime] = useState('09:00');
  const [participants, setParticipants] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) {
      return;
    }
    const layer = schedule?.layers[0];
    setName(schedule?.name ?? 'Primary');
    setTimezone(schedule?.timezone ?? 'UTC');
    setHandoffWeekday(layer?.handoff_weekday ?? 1);
    setHandoffTime(layer?.handoff_time ?? '09:00');
    setParticipants(layer?.participant_user_ids ?? []);
    setError(null);
  }, [open, schedule]);

  const orderedMembers = useMemo(
    () => [...members].sort((a, b) => {
      const aIndex = participants.indexOf(a.user_id);
      const bIndex = participants.indexOf(b.user_id);
      if (aIndex >= 0 || bIndex >= 0) {
        return (aIndex < 0 ? participants.length : aIndex) - (bIndex < 0 ? participants.length : bIndex);
      }
      return a.display_name.localeCompare(b.display_name);
    }),
    [members, participants],
  );

  const toggleParticipant = (userId: string) => {
    setParticipants((current) =>
      current.includes(userId) ? current.filter((id) => id !== userId) : [...current, userId],
    );
  };

  const moveParticipant = (userId: string, direction: -1 | 1) => {
    setParticipants((current) => {
      const index = current.indexOf(userId);
      if (index < 0) {
        return current;
      }
      const nextIndex = index + direction;
      if (nextIndex < 0 || nextIndex >= current.length) {
        return current;
      }
      const copy = [...current];
      [copy[index], copy[nextIndex]] = [copy[nextIndex], copy[index]];
      return copy;
    });
  };

  const handleSave = async () => {
    if (participants.length === 0) {
      setError(t('schedule.validation.participants'));
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onSave({ name, timezone, handoffWeekday, handoffTime, participants });
      onClose();
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : t('schedule.save_error'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      title={t(schedule ? 'schedule.edit' : 'schedule.create')}
      open={open}
      onClose={onClose}
      primaryLabel={t('schedule.save')}
      secondaryLabel={t('teams.cancel')}
      onPrimary={() => void handleSave()}
      primaryDisabled={saving}
      primaryLoading={saving}
    >
      <div className="space-y-4">
        <Input label={t('schedule.name_label')} value={name} onChange={setName} />
        <Input label={t('schedule.timezone_label')} value={timezone} onChange={setTimezone} />
        <Select label={t('schedule.handoff_weekday_label')} value={String(handoffWeekday)}
          options={WEEKDAYS.map((day) => ({ value: String(day), label: t(`schedule.weekday.${day}`) }))}
          onChange={(value) => setHandoffWeekday(Number(value))} />
        <Input label={t('schedule.handoff_time_label')} value={handoffTime} onChange={setHandoffTime} />
        <div>
          <p className="mb-2 text-sm font-medium text-zinc-700">{t('schedule.participants_label')}</p>
          <ul className="max-h-40 space-y-1 overflow-y-auto rounded-md border border-zinc-200 p-2 text-sm">
            {orderedMembers.map((member) => (
              <li key={member.user_id} className="flex items-center gap-2">
                <div className="min-w-0 flex-1">
                  <Checkbox id={`schedule-participant-${member.user_id}`} label={member.display_name}
                    checked={participants.includes(member.user_id)} onChange={() => toggleParticipant(member.user_id)} />
                </div>
                {participants.includes(member.user_id) ? (
                  <div className="flex gap-1">
                    <Button variant="ghost" aria-label={t('schedule.move_earlier', { name: member.display_name })} disabled={participants.indexOf(member.user_id) === 0} onClick={() => moveParticipant(member.user_id, -1)}>
                      ↑
                    </Button>
                    <Button variant="ghost" aria-label={t('schedule.move_later', { name: member.display_name })} disabled={participants.indexOf(member.user_id) === participants.length - 1} onClick={() => moveParticipant(member.user_id, 1)}>
                      ↓
                    </Button>
                  </div>
                ) : null}
              </li>
            ))}
          </ul>
        </div>
        {error ? <p role="alert" className="text-sm text-red-700">{error}</p> : null}
      </div>
    </Modal>
  );
}
