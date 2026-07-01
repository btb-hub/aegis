import type { CalendarOverride, CalendarSlot, OnCallUser } from './shiftsTypes';

const ROTATION = [
  { userId: 'user-alice', displayName: 'Alice', email: 'alice@example.com' },
  { userId: 'user-bob', displayName: 'Bob', email: 'bob@example.com' },
  { userId: 'user-carol', displayName: 'Carol', email: 'carol@example.com' },
];

function startOfMonthUTC(date: Date): Date {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), 1));
}

function addDaysUTC(date: Date, days: number): Date {
  const next = new Date(date);
  next.setUTCDate(next.getUTCDate() + days);
  return next;
}

function containsInstant(startAt: string, endAt: string, instant: Date): boolean {
  const start = new Date(startAt);
  const end = new Date(endAt);
  return start <= instant && instant < end;
}

export function buildDemoShiftsForMonth(month: Date = new Date()): {
  slots: CalendarSlot[];
  overrides: CalendarOverride[];
} {
  const first = startOfMonthUTC(month);
  const slots: CalendarSlot[] = [];

  let weekStart = first;
  for (let rotIndex = 0; rotIndex < 6; rotIndex += 1) {
    const weekEnd = addDaysUTC(weekStart, 7);
    const user = ROTATION[rotIndex % ROTATION.length];
    slots.push({
      id: `slot-${rotIndex + 1}`,
      userId: user.userId,
      displayName: user.displayName,
      startAt: weekStart.toISOString(),
      endAt: weekEnd.toISOString(),
      source: 'rotation',
    });
    weekStart = weekEnd;
  }

  const overrideStart = addDaysUTC(first, 14);
  const overrideEnd = addDaysUTC(overrideStart, 1);
  const overrides: CalendarOverride[] = [
    {
      id: 'override-1',
      userId: 'user-carol',
      displayName: 'Carol',
      startAt: overrideStart.toISOString(),
      endAt: overrideEnd.toISOString(),
    },
  ];

  return { slots, overrides };
}

export function resolveCurrentOnCall(
  slots: CalendarSlot[],
  overrides: CalendarOverride[],
  at: Date = new Date(),
): OnCallUser[] {
  const emails = Object.fromEntries(ROTATION.map((user) => [user.userId, user.email]));

  for (const override of overrides) {
    if (containsInstant(override.startAt, override.endAt, at)) {
      return [
        {
          userId: override.userId,
          displayName: override.displayName,
          email: emails[override.userId] ?? `${override.userId}@example.com`,
          source: 'override',
        },
      ];
    }
  }

  for (const slot of slots) {
    if (containsInstant(slot.startAt, slot.endAt, at)) {
      return [
        {
          userId: slot.userId,
          displayName: slot.displayName,
          email: emails[slot.userId] ?? `${slot.userId}@example.com`,
          source: slot.source,
        },
      ];
    }
  }

  return [];
}
