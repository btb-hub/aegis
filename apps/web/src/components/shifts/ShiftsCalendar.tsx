import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { formatDateTime } from '../../lib/formatDate';
import type { CalendarOverride, CalendarSlot } from '../../lib/shiftsTypes';

type ShiftsCalendarProps = {
  month: Date;
  slots: CalendarSlot[];
  overrides: CalendarOverride[];
};

type DayCell = {
  date: Date;
  inMonth: boolean;
  slots: CalendarSlot[];
  overrides: CalendarOverride[];
};

function startOfMonth(date: Date): Date {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), 1));
}

function addDays(date: Date, days: number): Date {
  const next = new Date(date);
  next.setUTCDate(next.getUTCDate() + days);
  return next;
}

function sameDay(a: Date, b: Date): boolean {
  return a.getUTCFullYear() === b.getUTCFullYear() && a.getUTCMonth() === b.getUTCMonth() && a.getUTCDate() === b.getUTCDate();
}

function overlapsDay(startAt: string, endAt: string, day: Date): boolean {
  const dayStart = new Date(Date.UTC(day.getUTCFullYear(), day.getUTCMonth(), day.getUTCDate()));
  const dayEnd = addDays(dayStart, 1);
  const start = new Date(startAt);
  const end = new Date(endAt);
  return start < dayEnd && end > dayStart;
}

function buildMonthGrid(month: Date): DayCell[] {
  const first = startOfMonth(month);
  const startOffset = first.getUTCDay();
  const gridStart = addDays(first, -startOffset);
  const cells: DayCell[] = [];

  for (let i = 0; i < 42; i += 1) {
    const date = addDays(gridStart, i);
    cells.push({
      date,
      inMonth: date.getUTCMonth() === month.getUTCMonth(),
      slots: [],
      overrides: [],
    });
  }
  return cells;
}

export function ShiftsCalendar({ month, slots, overrides }: ShiftsCalendarProps) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language;

  const cells = useMemo(() => {
    const grid = buildMonthGrid(month);
    for (const cell of grid) {
      cell.slots = slots.filter((slot) => overlapsDay(slot.startAt, slot.endAt, cell.date));
      cell.overrides = overrides.filter((override) => overlapsDay(override.startAt, override.endAt, cell.date));
    }
    return grid;
  }, [month, slots, overrides]);

  const monthLabel = formatDateTime(month, locale, { month: 'long', year: 'numeric' });
  const weekdays = [
    t('shifts.weekday.sun'),
    t('shifts.weekday.mon'),
    t('shifts.weekday.tue'),
    t('shifts.weekday.wed'),
    t('shifts.weekday.thu'),
    t('shifts.weekday.fri'),
    t('shifts.weekday.sat'),
  ];
  const today = new Date();

  return (
    <section aria-label={t('shifts.calendar_title')}>
      <h2 className="mb-4 text-xl font-semibold">{monthLabel}</h2>
      <div className="grid grid-cols-7 gap-px overflow-hidden rounded-lg border border-zinc-200 bg-zinc-200">
        {weekdays.map((label) => (
          <div key={label} className="bg-white px-2 py-2 text-center text-xs font-medium text-zinc-500">
            {label}
          </div>
        ))}
        {cells.map((cell) => {
          const isToday = sameDay(cell.date, today);
          return (
            <div
              key={cell.date.toISOString()}
              className={`min-h-24 bg-white p-2 ${cell.inMonth ? '' : 'bg-zinc-50 text-zinc-400'}`}
            >
              <div className={`text-sm font-medium ${isToday ? 'text-accent' : ''}`}>{cell.date.getUTCDate()}</div>
              <ul className="mt-1 space-y-1">
                {cell.slots.map((slot) => (
                  <li
                    key={slot.id}
                    className="truncate rounded bg-surface px-1 py-0.5 text-xs text-zinc-700"
                    title={slot.displayName}
                  >
                    {slot.displayName}
                  </li>
                ))}
                {cell.overrides.map((override) => (
                  <li
                    key={override.id}
                    className="truncate rounded bg-accent/10 px-1 py-0.5 text-xs font-medium text-accent"
                    title={override.displayName}
                  >
                    {override.displayName}
                  </li>
                ))}
              </ul>
            </div>
          );
        })}
      </div>
    </section>
  );
}
