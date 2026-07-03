import { describe, expect, it, vi } from 'vitest';
import { createSchedule, deleteOverride, fetchTeams, mapApiCalendarSlots, mapApiToOnCallUsers, monthRangeUTC } from './shiftsApi';

describe('shiftsApi', () => {
  it('computes month range in UTC', () => {
    const month = new Date('2026-06-15T12:00:00Z');
    const range = monthRangeUTC(month);
    expect(range.from).toBe('2026-06-01T00:00:00.000Z');
    expect(range.to).toBe('2026-06-30T23:59:59.000Z');
  });

  it('maps on-call users from API shape', () => {
    const users = mapApiToOnCallUsers([
      {
        user_id: 'u1',
        email: 'a@example.com',
        display_name: 'Alice',
        source: 'rotation',
      },
    ]);
    expect(users[0]).toEqual({
      userId: 'u1',
      email: 'a@example.com',
      displayName: 'Alice',
      source: 'rotation',
    });
  });

  it('splits calendar slots by source and enriches names', () => {
    const names = new Map([['u1', 'Alice'], ['u2', 'Bob']]);
    const mapped = mapApiCalendarSlots(
      [
        {
          id: 's1',
          team_id: 't1',
          user_id: 'u1',
          start_at: '2026-06-01T00:00:00Z',
          end_at: '2026-06-08T00:00:00Z',
          source: 'rotation',
        },
        {
          id: 'o1',
          team_id: 't1',
          user_id: 'u2',
          start_at: '2026-06-10T00:00:00Z',
          end_at: '2026-06-11T00:00:00Z',
          source: 'override',
        },
      ],
      names,
    );
    expect(mapped.slots).toHaveLength(1);
    expect(mapped.overrides).toHaveLength(1);
    expect(mapped.slots[0].displayName).toBe('Alice');
    expect(mapped.overrides[0].displayName).toBe('Bob');
  });

  it('creates schedule via API', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ id: 'sch-1', name: 'Primary', timezone: 'UTC', layers: [] }),
    }));
    const schedule = await createSchedule('team-1', {
      name: 'Primary',
      timezone: 'UTC',
      rotation: { handoff_weekday: 1, handoff_time: '09:00', participants: ['u1'] },
    });
    expect(schedule.id).toBe('sch-1');
    vi.unstubAllGlobals();
  });

  it('deletes override via API', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true }));
    await deleteOverride('team-1', 'override-1');
    expect(fetch).toHaveBeenCalled();
    vi.unstubAllGlobals();
  });

  it('fetches teams list', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ items: [{ id: 't1', name: 'Platform', description: '', created_at: '', updated_at: '' }] }),
    }));
    const teams = await fetchTeams();
    expect(teams).toHaveLength(1);
    vi.unstubAllGlobals();
  });
});
