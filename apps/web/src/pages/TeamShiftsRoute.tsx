import { useMemo, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Link, useParams } from 'react-router-dom';
import { OverrideFormModal } from '../components/shifts/OverrideFormModal';
import { ScheduleFormModal } from '../components/shifts/ScheduleFormModal';
import { Button } from '../components/ui/Button';
import { Toast } from '../components/ui/Toast';
import { useAuth } from '../context/AuthContext';
import {
  createOverride,
  createSchedule,
  deleteOverride,
  fetchCurrentOnCall,
  fetchOnCallCalendar,
  fetchTeam,
  fetchTeamMembers,
  fetchTeamOverrides,
  fetchTeamSchedules,
  mapApiCalendarSlots,
  mapApiToOnCallUsers,
  memberNameMap,
  monthRangeUTC,
  updateSchedule,
} from '../lib/shiftsApi';
import { apiFetch } from '../lib/apiClient';
import { queryKeys } from '../lib/queryClient';
import { useLoader } from '../lib/useLoader';
import type { CalendarOverride, CalendarSlot } from '../lib/shiftsTypes';
import type { TeamMember } from '../lib/teamTypes';
import { TeamShiftsPage } from './TeamShiftsPage';

const EMPTY_MEMBERS: TeamMember[] = [];

export function TeamShiftsRoute() {
  const { t } = useTranslation();
  const { teamId = '' } = useParams();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const queryClient = useQueryClient();
  const [month] = useState(() => new Date());
  const [scheduleModalOpen, setScheduleModalOpen] = useState(false);
  const [overrideModalOpen, setOverrideModalOpen] = useState(false);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);
  const [publishing, setPublishing] = useState(false);

  const shiftsQuery = useLoader(queryKeys.shifts.team(teamId), async () => {
    const range = monthRangeUTC(month);
    const [team, teamMembers, teamSchedules, teamOverrides, current] = await Promise.all([
      fetchTeam(teamId),
      fetchTeamMembers(teamId),
      fetchTeamSchedules(teamId),
      fetchTeamOverrides(teamId),
      fetchCurrentOnCall(teamId),
    ]);
    let slots: CalendarSlot[] = [];
    let calendarOverrides: CalendarOverride[] = [];
    if (teamSchedules.length > 0) {
      const calendar = await fetchOnCallCalendar(teamId, range.from, range.to);
      const mapped = mapApiCalendarSlots(calendar, memberNameMap(teamMembers));
      slots = mapped.slots;
      calendarOverrides = mapped.overrides;
    }
    return {
      teamName: team.name,
      canPublish: Boolean(team.express_chat_id || team.slack_channel_id),
      members: teamMembers,
      schedules: teamSchedules,
      overrides: teamOverrides,
      onCallUsers: mapApiToOnCallUsers(current),
      slots,
      calendarOverrides,
    };
  });

  const teamName = shiftsQuery.data?.teamName ?? '';
  const members = shiftsQuery.data?.members ?? EMPTY_MEMBERS;
  const schedules = shiftsQuery.data?.schedules ?? [];
  const overrides = shiftsQuery.data?.overrides ?? [];
  const onCallUsers = shiftsQuery.data?.onCallUsers ?? [];
  const slots = shiftsQuery.data?.slots ?? [];
  const calendarOverrides = shiftsQuery.data?.calendarOverrides ?? [];
  const loading = shiftsQuery.loading;
  const error = shiftsQuery.isError ? t('shifts.load_error') : null;
  const canPublish = shiftsQuery.data?.canPublish ?? false;

  const primarySchedule = schedules[0] ?? null;
  const nameByUserId = useMemo(() => memberNameMap(members), [members]);

  const refreshShifts = () => queryClient.invalidateQueries({ queryKey: queryKeys.shifts.team(teamId) });

  const saveSchedule = async (payload: {
    name: string;
    timezone: string;
    handoffWeekday: number;
    handoffTime: string;
    participants: string[];
  }) => {
    const body = {
      name: payload.name,
      timezone: payload.timezone,
      rotation: {
        handoff_weekday: payload.handoffWeekday,
        handoff_time: payload.handoffTime,
        participants: payload.participants,
      },
    };
    if (primarySchedule) {
      await updateSchedule(teamId, primarySchedule.id, body);
    } else {
      await createSchedule(teamId, body);
    }
    setToast({ message: t('schedule.saved'), variant: 'success' });
    await refreshShifts();
  };

  const addOverride = async (payload: { userId: string; startAt: string; endAt: string }) => {
    await createOverride(teamId, {
      user_id: payload.userId,
      start_at: payload.startAt,
      end_at: payload.endAt,
    });
    setToast({ message: t('override.saved'), variant: 'success' });
    await refreshShifts();
  };

  const removeOverride = async (overrideId: string) => {
    await deleteOverride(teamId, overrideId);
    setToast({ message: t('override.deleted'), variant: 'success' });
    await refreshShifts();
  };

  const publishOnCall = async () => {
    setPublishing(true);
    setToast(null);
    try {
      await apiFetch(`/api/v1/teams/${teamId}/on-call/publish`, { method: 'POST' });
      setToast({ message: t('shifts.published'), variant: 'success' });
    } catch (error) {
      const message = error instanceof Error ? error.message : t('shifts.publish_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setPublishing(false);
    }
  };

  if (loading) {
    return <p className="text-sm text-zinc-600">{t('shifts.loading')}</p>;
  }

  if (error) {
    return (
      <div className="space-y-3">
        <p className="text-sm text-red-700">{error}</p>
        <Button variant="secondary" onClick={() => void shiftsQuery.refetch()}>
          {t('shifts.retry')}
        </Button>
      </div>
    );
  }

  if (schedules.length === 0) {
    return (
      <div className="max-w-5xl space-y-4">
        <h1 className="text-3xl font-semibold">{teamName}</h1>
        <div className="rounded-lg border border-zinc-200 bg-surface px-4 py-6 text-sm text-zinc-700">
          <p>{t('shifts.no_schedule')}</p>
          <div className="mt-3 flex flex-wrap gap-3">
            <Link to={`/teams/${teamId}`} className="text-accent hover:underline">
              {t('shifts.no_schedule_cta')}
            </Link>
            {isAdmin ? (
              <Button onClick={() => setScheduleModalOpen(true)}>{t('schedule.create')}</Button>
            ) : null}
          </div>
        </div>
        {isAdmin ? (
          <ScheduleFormModal
            open={scheduleModalOpen}
            onClose={() => setScheduleModalOpen(false)}
            members={members}
            onSave={saveSchedule}
          />
        ) : null}
        {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {isAdmin ? (
        <div className="flex flex-wrap gap-2">
          <Button variant="secondary" onClick={() => setScheduleModalOpen(true)}>
            {primarySchedule ? t('schedule.edit') : t('schedule.create')}
          </Button>
          <Button variant="secondary" onClick={() => setOverrideModalOpen(true)}>
            {t('override.create')}
          </Button>
          {canPublish ? (
            <Button variant="secondary" disabled={publishing} onClick={() => void publishOnCall()}>
              {t('shifts.publish_now')}
            </Button>
          ) : null}
        </div>
      ) : null}
      <TeamShiftsPage
        teamName={teamName}
        onCallUsers={onCallUsers}
        slots={slots}
        overrides={calendarOverrides}
        month={month}
      />
      {isAdmin ? (
        <>
          <ScheduleFormModal
            open={scheduleModalOpen}
            onClose={() => setScheduleModalOpen(false)}
            members={members}
            schedule={primarySchedule}
            onSave={saveSchedule}
          />
          <OverrideFormModal
            open={overrideModalOpen}
            onClose={() => setOverrideModalOpen(false)}
            members={members}
            overrides={overrides}
            nameByUserId={nameByUserId}
            onCreate={addOverride}
            onDelete={removeOverride}
          />
        </>
      ) : null}
      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </div>
  );
}
