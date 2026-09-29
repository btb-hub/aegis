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

  return (
    <section aria-label={t('shifts.calendar_title')} className="overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-zinc-200 px-5 py-4">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className="text-[17px] font-semibold tracking-tight text-zinc-900">{t('shifts.on_call_title')}</h2>
          <div className="flex items-center gap-1.5">
            <button type="button" aria-label={t('shifts.previous_period')} onClick={() => move(-1)} className="flex h-7 w-7 items-center justify-center rounded-md border border-zinc-200 text-zinc-500 hover:bg-zinc-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent">‹</button>
            <span className="min-w-44 px-1 text-center font-mono text-xs font-medium text-zinc-700">{view === 'week' ? weekLabel : monthLabel}</span>
            <button type="button" aria-label={t('shifts.next_period')} onClick={() => move(1)} className="flex h-7 w-7 items-center justify-center rounded-md border border-zinc-200 text-zinc-500 hover:bg-zinc-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent">›</button>
          </div>
          <button type="button" onClick={goToday} className="rounded-md border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent">{t('shifts.today')}</button>
        </div>
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-3 font-mono text-[11px] text-zinc-500">
            <span className="inline-flex items-center gap-1.5"><span className="h-3 w-3 rounded-[3px] border border-blue-300 bg-blue-100" />{t('shifts.rotation')}</span>
            <span className="inline-flex items-center gap-1.5"><span className="h-3 w-3 rounded-[3px] border border-dashed border-severity-p3 bg-amber-50" />{t('shifts.override')}</span>
            <span className="inline-flex items-center gap-1.5"><span className="h-3 w-0.5 bg-accent" />{t('shifts.now')}</span>
          </div>
          <div className="inline-flex overflow-hidden rounded-md border border-zinc-300 text-xs">
            <button type="button" aria-pressed={view === 'week'} onClick={() => setView('week')} className={`px-3 py-1.5 font-medium ${view === 'week' ? 'bg-zinc-900 text-white' : 'text-zinc-500 hover:bg-zinc-50'}`}>{t('shifts.week')}</button>
            <button type="button" aria-pressed={view === 'month'} onClick={() => setView('month')} className={`px-3 py-1.5 font-medium ${view === 'month' ? 'bg-zinc-900 text-white' : 'text-zinc-500 hover:bg-zinc-50'}`}>{t('shifts.month')}</button>
          </div>
          {onAddOverride ? <button type="button" onClick={onAddOverride} className="rounded-md bg-accent px-3.5 py-1.5 text-xs font-medium text-white hover:bg-blue-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent">{t('override.create')}</button> : null}
        </div>
      </div>
      {view === 'week' ? (
        <div className="overflow-x-auto">
          <div className="relative min-w-[880px]">
            <div className="grid grid-cols-[160px_repeat(7,minmax(0,1fr))] border-b border-zinc-200">
              <div className="flex h-[46px] items-center px-4 font-mono text-[10px] font-semibold uppercase tracking-wider text-zinc-400">{t('shifts.calendar_team')}</div>
              {days.map((day) => {
                const isToday = day.getTime() === today.getTime();
                return <div key={day.toISOString()} className={`flex flex-col items-center justify-center border-l border-zinc-100 ${isToday ? 'bg-blue-50' : day.getUTCDay() === 0 || day.getUTCDay() === 6 ? 'bg-zinc-50' : ''}`}>
                  <span className={`font-mono text-[10px] font-semibold uppercase tracking-wider ${isToday ? 'text-accent' : 'text-zinc-400'}`}>{weekdayFormatter.format(day)}</span>
                  <span className={`text-[13px] ${isToday ? 'font-semibold text-blue-700' : 'text-zinc-600'}`}>{day.getUTCDate()}</span>
                </div>;
              })}
            </div>
            {activeLanes.map((lane) => (
              <div key={lane.id} className="grid grid-cols-[160px_minmax(0,1fr)] border-b border-zinc-100 last:border-b-0">
                <div className="flex h-[66px] flex-col justify-center px-4">
                  {lane.href ? <a href={lane.href} className="truncate text-[13px] font-semibold text-zinc-900 hover:text-accent">{lane.name}</a> : <span className="truncate text-[13px] font-semibold text-zinc-900">{lane.name}</span>}
                  <span className="font-mono text-[10px] text-zinc-400">{t('shifts.rotation')}</span>
                </div>
                <div className="relative h-[66px]">
                  <div className="absolute inset-0 grid grid-cols-7">{days.map((day) => <div key={day.toISOString()} className={`border-l border-zinc-100 ${day.getUTCDay() === 0 || day.getUTCDay() === 6 ? 'bg-zinc-50' : day.getTime() === today.getTime() ? 'bg-blue-50/40' : ''}`} />)}</div>
                  {lane.slots.map((slot) => {
                    const style = overlapStyle(slot.startAt, slot.endAt, weekStart);
                    return style ? <div key={slot.id} title={`${slot.displayName} · ${t('shifts.rotation')}`} style={style} className="absolute top-2 z-10 flex h-[48px] items-center gap-2 overflow-hidden rounded-[7px] border border-blue-200 bg-blue-50 px-2.5 text-[12px] font-semibold text-blue-900">
                      <span className="flex h-[22px] w-[22px] shrink-0 items-center justify-center rounded-full bg-blue-100 text-[10px] text-blue-700">{initials(slot.displayName)}</span><span className="truncate">{slot.displayName}</span>
                    </div> : null;
                  })}
                  {lane.overrides.map((override) => {
                    const style = overlapStyle(override.startAt, override.endAt, weekStart);
                    return style ? <div key={override.id} title={`${override.displayName} · ${t('shifts.override')}`} style={style} className="absolute top-2 z-20 flex h-[48px] min-w-[54px] flex-col items-center justify-center overflow-hidden rounded-[7px] border border-dashed border-severity-p3 bg-amber-50 px-1 text-center text-amber-900">
                      <span className="font-mono text-[9px] font-semibold uppercase tracking-wide text-amber-700">{t('shifts.override')}</span><span className="truncate text-[11px] font-semibold">{override.displayName}</span>
                    </div> : null;
                  })}
                </div>
              </div>
            ))}
            {nowInWeek ? <div aria-hidden="true" className="pointer-events-none absolute inset-y-0 left-40 right-0 z-30">
              <div className="absolute inset-y-0 w-0.5 bg-accent" style={{ left: nowPosition }}>
                <span className="absolute left-1/2 top-1 -translate-x-1/2 whitespace-nowrap rounded bg-accent px-1.5 py-0.5 font-mono text-[9px] font-semibold text-white">{t('shifts.now').toUpperCase()} {new Intl.DateTimeFormat(locale, { hour: '2-digit', minute: '2-digit' }).format(now)}</span>
                <span className="absolute -left-[3px] top-[42px] h-2 w-2 rounded-full border-2 border-white bg-accent" />
              </div>
            </div> : null}
          </div>
        </div>
      ) : (
        <div className="overflow-x-auto p-4">
          <div className="grid min-w-[700px] grid-cols-7 gap-px overflow-hidden rounded-md border border-zinc-200 bg-zinc-200">
            {days.map((day) => <div key={day.toISOString()} className="bg-zinc-50 px-2 py-2 text-center font-mono text-[10px] font-semibold uppercase tracking-wider text-zinc-400">{weekdayFormatter.format(day)}</div>)}
            {monthDays.map((day) => {
              const entries = activeLanes.flatMap((lane) => [...lane.slots.map((slot) => ({ ...slot, kind: 'rotation' })), ...lane.overrides.map((override) => ({ ...override, kind: 'override' }))].filter((entry) => new Date(entry.startAt) < addDays(day, 1) && new Date(entry.endAt) > day).map((entry) => ({ ...entry, team: lane.name })));
              return <div key={day.toISOString()} className={`min-h-24 bg-white p-2 ${day.getUTCMonth() !== anchor.getUTCMonth() ? 'bg-zinc-50 text-zinc-400' : ''}`}>
                <div className={`font-mono text-xs ${day.getTime() === today.getTime() ? 'font-semibold text-accent' : ''}`}>{day.getUTCDate()}</div>
                <ul className="mt-1 space-y-1">{entries.map((entry) => <li key={`${entry.team}-${entry.kind}-${entry.id}`} className={`truncate rounded px-1 py-0.5 text-[11px] ${entry.kind === 'override' ? 'bg-amber-50 text-amber-800' : 'bg-blue-50 text-blue-800'}`} title={`${entry.team}: ${entry.displayName}`}>{activeLanes.length > 1 ? `${entry.team}: ` : ''}{entry.displayName}</li>)}</ul>
              </div>;
            })}
          </div>
        </div>
      )}
    </section>
  );
}
