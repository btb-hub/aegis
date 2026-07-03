import { useCallback, useEffect, useMemo, useState } from 'react';
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
  type ApiSchedule,
} from '../lib/shiftsApi';
import type { CalendarOverride, CalendarSlot, OnCallUser } from '../lib/shiftsTypes';
import type { TeamMember } from '../lib/teamTypes';
import { TeamShiftsPage } from './TeamShiftsPage';

export function TeamShiftsRoute() {
  const { t } = useTranslation();
  const { teamId = '' } = useParams();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const [month] = useState(() => new Date());
  const [teamName, setTeamName] = useState('');
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [schedules, setSchedules] = useState<ApiSchedule[]>([]);
  const [overrides, setOverrides] = useState<Awaited<ReturnType<typeof fetchTeamOverrides>>>([]);
  const [onCallUsers, setOnCallUsers] = useState<OnCallUser[]>([]);
  const [slots, setSlots] = useState<CalendarSlot[]>([]);
  const [calendarOverrides, setCalendarOverrides] = useState<CalendarOverride[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [scheduleModalOpen, setScheduleModalOpen] = useState(false);
  const [overrideModalOpen, setOverrideModalOpen] = useState(false);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);

  const primarySchedule = schedules[0] ?? null;
  const nameByUserId = useMemo(() => memberNameMap(members), [members]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [team, teamMembers, teamSchedules, teamOverrides, current, range] = await Promise.all([
        fetchTeam(teamId),
        fetchTeamMembers(teamId),
        fetchTeamSchedules(teamId),
        fetchTeamOverrides(teamId),
        fetchCurrentOnCall(teamId),
        Promise.resolve(monthRangeUTC(month)),
      ]);
      setTeamName(team.name);
      setMembers(teamMembers);
      setSchedules(teamSchedules);
      setOverrides(teamOverrides);
      setOnCallUsers(mapApiToOnCallUsers(current));

      if (teamSchedules.length === 0) {
        setSlots([]);
        setCalendarOverrides([]);
        return;
      }

      const calendar = await fetchOnCallCalendar(teamId, range.from, range.to);
      const mapped = mapApiCalendarSlots(calendar, memberNameMap(teamMembers));
      setSlots(mapped.slots);
      setCalendarOverrides(mapped.overrides);
    } catch {
      setError(t('shifts.load_error'));
      setTeamName('');
      setMembers([]);
      setSchedules([]);
      setOverrides([]);
      setOnCallUsers([]);
      setSlots([]);
      setCalendarOverrides([]);
    } finally {
      setLoading(false);
    }
  }, [teamId, month, t]);

  const refreshCalendar = useCallback(async () => {
    const range = monthRangeUTC(month);
    const [current, calendar] = await Promise.all([
      fetchCurrentOnCall(teamId),
      fetchOnCallCalendar(teamId, range.from, range.to),
    ]);
    setOnCallUsers(mapApiToOnCallUsers(current));
    const mapped = mapApiCalendarSlots(calendar, memberNameMap(members));
    setSlots(mapped.slots);
    setCalendarOverrides(mapped.overrides);
  }, [members, month, teamId]);

  useEffect(() => {
    void load();
  }, [load]);

  const saveSchedule = useCallback(
    async (payload: {
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
      const saved = primarySchedule
        ? await updateSchedule(teamId, primarySchedule.id, body)
        : await createSchedule(teamId, body);
      setSchedules((current) => {
        const index = current.findIndex((schedule) => schedule.id === saved.id);
        if (index >= 0) {
          const next = [...current];
          next[index] = saved;
          return next;
        }
        return [...current, saved];
      });
      setToast({ message: t('schedule.saved'), variant: 'success' });
      await refreshCalendar();
    },
    [primarySchedule, refreshCalendar, t, teamId],
  );

  const addOverride = useCallback(
    async (payload: { userId: string; startAt: string; endAt: string }) => {
      await createOverride(teamId, {
        user_id: payload.userId,
        start_at: payload.startAt,
        end_at: payload.endAt,
      });
      setToast({ message: t('override.saved'), variant: 'success' });
      await refreshCalendar();
      const teamOverrides = await fetchTeamOverrides(teamId);
      setOverrides(teamOverrides);
    },
    [refreshCalendar, t, teamId],
  );

  const removeOverride = useCallback(
    async (overrideId: string) => {
      await deleteOverride(teamId, overrideId);
      setToast({ message: t('override.deleted'), variant: 'success' });
      await refreshCalendar();
      const teamOverrides = await fetchTeamOverrides(teamId);
      setOverrides(teamOverrides);
    },
    [refreshCalendar, t, teamId],
  );

  if (loading) {
    return <p className="text-sm text-zinc-600">{t('shifts.loading')}</p>;
  }

  if (error) {
    return (
      <div className="space-y-3">
        <p className="text-sm text-red-700">{error}</p>
        <Button variant="secondary" onClick={() => void load()}>
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
