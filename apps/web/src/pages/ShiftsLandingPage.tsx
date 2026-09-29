import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, Navigate } from 'react-router-dom';
import { Button } from '../components/ui/Button';
import { ShiftsCalendar } from '../components/shifts/ShiftsCalendar';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { calendarRangeUTC, fetchOnCallCalendar, fetchTeamMembers, fetchTeams, mapApiCalendarSlots, memberNameMap } from '../lib/shiftsApi';
import type { CalendarOverride, CalendarSlot } from '../lib/shiftsTypes';
import type { Team } from '../lib/teamTypes';

export function ShiftsLandingPage() {
  const { t } = useTranslation();
  const [teams, setTeams] = useState<Team[] | null>(null);
  const [date, setDate] = useState(() => new Date());
  const [lanes, setLanes] = useState<Array<{ id: string; name: string; href: string; slots: CalendarSlot[]; overrides: CalendarOverride[] }>>([]);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const items = await fetchTeams();
      setTeams(items);
      if (items.length > 1) {
        const range = calendarRangeUTC(date);
        const loaded = await Promise.all(items.map(async (team) => {
          try {
            const [members, calendar] = await Promise.all([
              fetchTeamMembers(team.id),
              fetchOnCallCalendar(team.id, range.from, range.to),
            ]);
            return { id: team.id, name: team.name, href: `/teams/${team.id}/shifts`, ...mapApiCalendarSlots(calendar, memberNameMap(members)) };
          } catch {
            return { id: team.id, name: team.name, href: `/teams/${team.id}/shifts`, slots: [], overrides: [] };
          }
        }));
        setLanes(loaded);
      }
    } catch {
      setError(t('shifts.load_error'));
      setTeams([]);
    }
  }, [date, t]);

  useEffect(() => {
    void load();
  }, [load]);

  if (teams === null) {
    return <p className="text-sm text-zinc-600">{t('shifts.loading')}</p>;
  }

  if (error) {
    return (
      <PageContent>
        <p className="text-sm text-red-700">{error}</p>
        <Button variant="secondary" onClick={() => void load()}>
          {t('shifts.retry')}
        </Button>
      </PageContent>
    );
  }

  if (teams.length === 1) {
    return <Navigate to={`/teams/${teams[0].id}/shifts`} replace />;
  }

  if (teams.length === 0) {
    return (
      <PageContent>
        <PageHeader
          title={t('nav.shifts')}
          subtitle={t('shifts.no_teams')}
          breadcrumb={{
            ariaLabel: t('nav.breadcrumb_label'),
            items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('nav.shifts') }],
          }}
        />
        <Link to="/teams" className="text-sm text-accent hover:underline">
          {t('shifts.no_teams_cta')}
        </Link>
      </PageContent>
    );
  }

  return (
    <PageContent>
      <PageHeader
        title={t('nav.shifts')}
        subtitle={t('shifts.select_team')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('nav.shifts') }],
        }}
      />
      <ShiftsCalendar month={date} lanes={lanes.length ? lanes : teams.map((team) => ({ id: team.id, name: team.name, href: `/teams/${team.id}/shifts`, slots: [], overrides: [] }))} onDateChange={setDate} />
    </PageContent>
  );
}
