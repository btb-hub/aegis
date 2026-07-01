import { describe, expect, it } from 'vitest';
import { buildDemoShiftsForMonth, resolveCurrentOnCall } from './shiftsDemoData';

describe('buildDemoShiftsForMonth', () => {
  it('generates weekly rotation slots for the requested month', () => {
    const month = new Date('2026-06-01T00:00:00Z');
    const { slots, overrides } = buildDemoShiftsForMonth(month);

    expect(slots.length).toBeGreaterThanOrEqual(4);
    expect(overrides).toHaveLength(1);
    expect(slots.some((slot) => slot.displayName === 'Bob')).toBe(true);
  });
});

describe('resolveCurrentOnCall', () => {
  it('returns the user covering the given instant from rotation slots', () => {
    const month = new Date('2026-06-01T00:00:00Z');
    const { slots, overrides } = buildDemoShiftsForMonth(month);
    const at = new Date('2026-06-30T12:00:00Z');

    const users = resolveCurrentOnCall(slots, overrides, at);

    expect(users).toHaveLength(1);
    expect(users[0]?.displayName).toBe('Bob');
  });

  it('prefers overrides over rotation slots', () => {
    const month = new Date('2026-06-01T00:00:00Z');
    const { slots, overrides } = buildDemoShiftsForMonth(month);
    const at = new Date('2026-06-15T12:00:00Z');

    const users = resolveCurrentOnCall(slots, overrides, at);

    expect(users[0]?.displayName).toBe('Carol');
    expect(users[0]?.source).toBe('override');
  });

  it('returns empty when no slot or override covers the instant', () => {
    const month = new Date('2026-06-01T00:00:00Z');
    const { slots, overrides } = buildDemoShiftsForMonth(month);
    const at = new Date('2026-07-15T12:00:00Z');

    expect(resolveCurrentOnCall(slots, overrides, at)).toEqual([]);
  });

  it('covers the first day of the month from rotation start', () => {
    const month = new Date('2026-07-01T00:00:00Z');
    const { slots, overrides } = buildDemoShiftsForMonth(month);
    const at = new Date('2026-07-01T12:00:00Z');

    const users = resolveCurrentOnCall(slots, overrides, at);

    expect(users).toHaveLength(1);
    expect(users[0]?.displayName).toBe('Alice');
  });
});
