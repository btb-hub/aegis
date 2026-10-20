import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { CalendarOverride, CalendarSlot } from '../../lib/shiftsTypes';

type CalendarLane = {
  id: string;
  name: string;
  href?: string;
  slots: CalendarSlot[];
  overrides: CalendarOverride[];
};

type ShiftsCalendarProps = {
  month: Date;
  slots?: CalendarSlot[];
  overrides?: CalendarOverride[];
  teamName?: string;
  lanes?: CalendarLane[];
  onDateChange?: (date: Date) => void;
  onAddOverride?: () => void;
};

const DAY_MS = 86_400_000;

function startOfDay(date: Date) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()));
}

function addDays(date: Date, days: number) {
  return new Date(date.getTime() + days * DAY_MS);
}

function startOfWeek(date: Date) {
  const day = startOfDay(date);
  return addDays(day, -(day.getUTCDay() + 6) % 7);
}

function initials(name: string) {
  return (name ?? '').split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]).join('').toUpperCase();
}

function overlapStyle(startAt: string, endAt: string, weekStart: Date) {
  const start = Math.max(new Date(startAt).getTime(), weekStart.getTime());
  const end = Math.min(new Date(endAt).getTime(), weekStart.getTime() + 7 * DAY_MS);
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) {
    return null;
  }
  return { left: `${(start - weekStart.getTime()) / (7 * DAY_MS) * 100}%`, width: `${(end - start) / (7 * DAY_MS) * 100}%` };
}

export function ShiftsCalendar({ month, slots = [], overrides = [], teamName, lanes, onDateChange, onAddOverride }: ShiftsCalendarProps) {
  const { t, i18n } = useTranslation();
  const [internalDate, setInternalDate] = useState(month);
  const [view, setView] = useState<'week' | 'month'>('week');
  const anchor = onDateChange ? month : internalDate;
  const weekStart = startOfWeek(anchor);
  const days = Array.from({ length: 7 }, (_, index) => addDays(weekStart, index));
  const now = new Date();
  const today = startOfDay(now);
  const nowInWeek = now >= weekStart && now < addDays(weekStart, 7);
  const nowPosition = `${(now.getTime() - weekStart.getTime()) / (7 * DAY_MS) * 100}%`;
  const activeLanes = lanes ?? [{ id: 'team', name: teamName ?? t('shifts.calendar_team'), slots, overrides }];
  const locale = i18n.language === 'ru' ? 'ru-RU' : 'en-US';
  const monthFormatter = new Intl.DateTimeFormat(locale, { month: 'short', timeZone: 'UTC' });
  const weekdayFormatter = new Intl.DateTimeFormat(locale, { weekday: 'short', timeZone: 'UTC' });
  const monthLabel = new Intl.DateTimeFormat(locale, { month: 'long', year: 'numeric', timeZone: 'UTC' }).format(anchor);
  const weekLabel = `${monthFormatter.format(days[0])} ${days[0].getUTCDate()} – ${monthFormatter.format(days[6])} ${days[6].getUTCDate()}, ${days[6].getUTCFullYear()}`;

  function move(direction: number) {
    const next = view === 'week' ? addDays(anchor, direction * 7) : new Date(Date.UTC(anchor.getUTCFullYear(), anchor.getUTCMonth() + direction, Math.min(anchor.getUTCDate(), 28)));
    if (onDateChange) {
      onDateChange(next);
    } else {
      setInternalDate(next);
    }
  }

  function goToday() {
    const next = new Date();
    if (onDateChange) {
      onDateChange(next);
    } else {
      setInternalDate(next);
    }
  }

  const firstOfMonth = new Date(Date.UTC(anchor.getUTCFullYear(), anchor.getUTCMonth(), 1));
  const monthStart = startOfWeek(firstOfMonth);
  const monthDays = Array.from({ length: 42 }, (_, index) => addDays(monthStart, index));

  const agendaDays = view === 'week' ? days : monthDays.filter((day) => day.getUTCMonth() === anchor.getUTCMonth());
  const dateFormatter = new Intl.DateTimeFormat(locale, { weekday: 'short', month: 'short', day: 'numeric', timeZone: 'UTC' });
  const timeFormatter = new Intl.DateTimeFormat(locale, { hour: '2-digit', minute: '2-digit', hourCycle: 'h23', timeZone: 'UTC' });
  const entriesForDay = (day: Date) => activeLanes.flatMap((lane) => [
    ...lane.overrides.map((entry) => ({ ...entry, kind: 'override' as const, team: lane.name })),
    ...lane.slots.map((entry) => ({ ...entry, kind: 'rotation' as const, team: lane.name })),
  ].filter((entry) => new Date(entry.startAt) < addDays(day, 1) && new Date(entry.endAt) > day));
  const dayTime = (entry: { startAt: string; endAt: string }, day: Date) => {
    const from = Math.max(new Date(entry.startAt).getTime(), day.getTime());
    const to = Math.min(new Date(entry.endAt).getTime(), addDays(day, 1).getTime());
    return `${timeFormatter.format(from)}–${to === addDays(day, 1).getTime() ? '24:00' : timeFormatter.format(to)}`;
  };

  return (
    <section aria-label={t('shifts.calendar_title')} className="overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-zinc-200 px-[16px] py-4 sm:px-5">
        <div className="flex min-w-0 flex-wrap items-center gap-3">
          <h2 className="text-[17px] font-semibold tracking-tight text-zinc-900">{t('shifts.on_call_title')}</h2>
          <div className="grid min-w-0 max-w-full grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-1.5">
            <button type="button" aria-label={t('shifts.previous_period')} onClick={() => move(-1)} className="flex h-[44px] w-[44px] shrink-0 sm:h-7 sm:w-7 items-center justify-center rounded-md border border-zinc-200 text-zinc-500 hover:bg-zinc-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent">‹</button>
            <span className="min-w-0 break-words px-1 text-center text-sm font-medium text-zinc-700">{view === 'week' ? weekLabel : monthLabel}</span>
            <button type="button" aria-label={t('shifts.next_period')} onClick={() => move(1)} className="flex h-[44px] w-[44px] shrink-0 sm:h-7 sm:w-7 items-center justify-center rounded-md border border-zinc-200 text-zinc-500 hover:bg-zinc-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent">›</button>
          </div>
          <button type="button" onClick={goToday} className="rounded-md border border-zinc-300 px-3 py-2 text-sm min-h-11 sm:min-h-8 font-medium text-zinc-700 hover:bg-zinc-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent">{t('shifts.today')}</button>
        </div>
        <div className="flex min-w-0 max-w-full flex-wrap items-center gap-4">
          <div className="hidden items-center gap-3 font-mono text-sm text-zinc-600 sm:flex">
            <span className="inline-flex items-center gap-1.5"><span className="h-3 w-3 rounded-[3px] border border-blue-300 bg-blue-100" />{t('shifts.rotation')}</span>
            <span className="inline-flex items-center gap-1.5"><span className="h-3 w-3 rounded-[3px] border border-dashed border-severity-p3 bg-amber-50" />{t('shifts.override')}</span>
            <span className="inline-flex items-center gap-1.5"><span className="h-3 w-0.5 bg-accent" />{t('shifts.now')}</span>
          </div>
          <div className="inline-flex max-w-full flex-wrap overflow-hidden rounded-md border border-zinc-300 text-sm">
            <button type="button" aria-pressed={view === 'week'} onClick={() => setView('week')} className={`min-h-11 px-3 py-2 font-medium sm:min-h-8 sm:py-1.5 ${view === 'week' ? 'bg-zinc-900 text-white' : 'text-zinc-500 hover:bg-zinc-50'}`}>{t('shifts.week')}</button>
            <button type="button" aria-pressed={view === 'month'} onClick={() => setView('month')} className={`min-h-11 px-3 py-2 font-medium sm:min-h-8 sm:py-1.5 ${view === 'month' ? 'bg-zinc-900 text-white' : 'text-zinc-500 hover:bg-zinc-50'}`}>{t('shifts.month')}</button>
          </div>
          {onAddOverride ? <button type="button" onClick={onAddOverride} className="max-w-full break-words rounded-md min-h-11 bg-accent px-3.5 py-2 text-sm sm:min-h-8 sm:py-1.5 font-medium text-white hover:bg-blue-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent">{t('override.create')}</button> : null}
        </div>
      </div>
      <section aria-label={t('shifts.agenda')} className="sm:hidden">
        <p className="border-b border-zinc-200 px-4 py-2 text-xs text-zinc-600">{t('shifts.timezone')}</p>
        <ol className="divide-y divide-zinc-200">
          {agendaDays.map((day) => {
            const entries = entriesForDay(day);
            return <li key={day.toISOString()} className="space-y-2 px-4 py-3">
              <h3 className="text-sm font-semibold text-zinc-800"><time dateTime={day.toISOString()}>{dateFormatter.format(day)}</time></h3>
              {entries.length ? <ul className="space-y-2">{entries.map((entry) => <li key={`${entry.team}-${entry.kind}-${entry.id}`} className={`min-w-0 border-l-2 pl-3 ${entry.kind === 'override' ? 'border-severity-p3' : 'border-accent'}`}>
                <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-sm"><span className="break-words font-medium text-zinc-900">{entry.displayName}</span><span className="tabular-nums text-zinc-600">{dayTime(entry, day)}</span></div>
                <p className="text-xs text-zinc-600">{activeLanes.length > 1 ? `${entry.team} · ` : ''}{t(`shifts.${entry.kind}`)}</p>
              </li>)}</ul> : <p className="text-sm text-zinc-600">{t('shifts.day_empty')}</p>}
            </li>;
          })}
        </ol>
      </section>
      <div className="hidden sm:block">
      {view === 'week' ? (
        <div className="overflow-x-auto">
          <div className="relative min-w-[880px]">
            <div className="grid grid-cols-[160px_repeat(7,minmax(0,1fr))] border-b border-zinc-200">
              <div className="flex h-[46px] items-center px-4 font-mono text-xs font-semibold uppercase tracking-wider text-zinc-600">{t('shifts.calendar_team')}</div>
              {days.map((day) => {
                const isToday = day.getTime() === today.getTime();
                return <div key={day.toISOString()} className={`flex flex-col items-center justify-center border-l border-zinc-100 ${isToday ? 'bg-blue-50' : day.getUTCDay() === 0 || day.getUTCDay() === 6 ? 'bg-zinc-50' : ''}`}>
                  <span className={`font-mono text-xs font-semibold uppercase tracking-wider ${isToday ? 'text-accent' : 'text-zinc-600'}`}>{weekdayFormatter.format(day)}</span>
                  <span className={`text-sm ${isToday ? 'font-semibold text-blue-700' : 'text-zinc-600'}`}>{day.getUTCDate()}</span>
                </div>;
              })}
            </div>
            {activeLanes.map((lane) => (
              <div key={lane.id} className="grid grid-cols-[160px_minmax(0,1fr)] border-b border-zinc-100 last:border-b-0">
                <div className="flex h-[66px] flex-col justify-center px-4">
                  {lane.href ? <a href={lane.href} className="truncate text-sm font-semibold text-zinc-900 hover:text-accent">{lane.name}</a> : <span className="truncate text-sm font-semibold text-zinc-900">{lane.name}</span>}
                  <span className="font-mono text-xs text-zinc-600">{t('shifts.rotation')}</span>
                </div>
                <div className="relative h-[66px]">
                  <div className="absolute inset-0 grid grid-cols-7">{days.map((day) => <div key={day.toISOString()} className={`border-l border-zinc-100 ${day.getUTCDay() === 0 || day.getUTCDay() === 6 ? 'bg-zinc-50' : day.getTime() === today.getTime() ? 'bg-blue-50/40' : ''}`} />)}</div>
                  {lane.slots.map((slot) => {
                    const style = overlapStyle(slot.startAt, slot.endAt, weekStart);
                    return style ? <div key={slot.id} title={`${slot.displayName} · ${t('shifts.rotation')}`} style={style} className="absolute top-2 z-10 flex h-[48px] items-center gap-2 overflow-hidden rounded-[7px] border border-blue-200 bg-blue-50 px-2.5 text-sm font-semibold text-blue-900">
                      <span className="flex h-[22px] w-[22px] shrink-0 items-center justify-center rounded-full bg-blue-100 text-xs text-blue-700">{initials(slot.displayName)}</span><span className="truncate">{slot.displayName}</span>
                    </div> : null;
                  })}
                  {lane.overrides.map((override) => {
                    const style = overlapStyle(override.startAt, override.endAt, weekStart);
                    return style ? <div key={override.id} title={`${override.displayName} · ${t('shifts.override')}`} style={style} className="absolute top-2 z-20 flex h-[48px] min-w-[54px] flex-col items-center justify-center overflow-hidden rounded-[7px] border border-dashed border-severity-p3 bg-amber-50 px-1 text-center text-amber-900">
                      <span className="font-mono text-xs font-semibold uppercase tracking-wide text-amber-700">{t('shifts.override')}</span><span className="truncate text-sm font-semibold">{override.displayName}</span>
                    </div> : null;
                  })}
                </div>
              </div>
            ))}
            {nowInWeek ? <div aria-hidden="true" className="pointer-events-none absolute inset-y-0 left-40 right-0 z-30">
              <div className="absolute inset-y-0 w-0.5 bg-accent" style={{ left: nowPosition }}>
                <span className="absolute left-1/2 top-1 -translate-x-1/2 whitespace-nowrap rounded bg-accent px-1.5 py-0.5 font-mono text-xs font-semibold text-white">{t('shifts.now').toUpperCase()} {new Intl.DateTimeFormat(locale, { hour: '2-digit', minute: '2-digit', timeZone: 'UTC' }).format(now)}</span>
                <span className="absolute -left-[3px] top-[42px] h-2 w-2 rounded-full border-2 border-white bg-accent" />
              </div>
            </div> : null}
          </div>
        </div>
      ) : (
        <div className="overflow-x-auto p-4">
          <div className="grid min-w-[700px] grid-cols-7 gap-px overflow-hidden rounded-md border border-zinc-200 bg-zinc-200">
            {days.map((day) => <div key={day.toISOString()} className="bg-zinc-50 px-2 py-2 text-center font-mono text-xs font-semibold uppercase tracking-wider text-zinc-600">{weekdayFormatter.format(day)}</div>)}
            {monthDays.map((day) => {
              const entries = activeLanes.flatMap((lane) => [...lane.slots.map((slot) => ({ ...slot, kind: 'rotation' })), ...lane.overrides.map((override) => ({ ...override, kind: 'override' }))].filter((entry) => new Date(entry.startAt) < addDays(day, 1) && new Date(entry.endAt) > day).map((entry) => ({ ...entry, team: lane.name })));
              return <div key={day.toISOString()} className={`min-h-24 bg-white p-2 ${day.getUTCMonth() !== anchor.getUTCMonth() ? 'bg-zinc-50 text-zinc-600' : ''}`}>
                <div className={`font-mono text-xs ${day.getTime() === today.getTime() ? 'font-semibold text-accent' : ''}`}>{day.getUTCDate()}</div>
                <ul className="mt-1 space-y-1">{entries.map((entry) => <li key={`${entry.team}-${entry.kind}-${entry.id}`} className={`truncate rounded px-1 py-0.5 text-sm ${entry.kind === 'override' ? 'bg-amber-50 text-amber-800' : 'bg-blue-50 text-blue-800'}`} title={`${entry.team}: ${entry.displayName}`}>{activeLanes.length > 1 ? `${entry.team}: ` : ''}{entry.displayName}</li>)}</ul>
              </div>;
            })}
          </div>
        </div>
      )}
      </div>
    </section>
  );
}
