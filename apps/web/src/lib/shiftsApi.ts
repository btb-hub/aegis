import type { CalendarOverride, CalendarSlot, OnCallUser } from './shiftsTypes';
import type { Team, TeamMember } from './teamTypes';

export type ApiOnCallUser = {
  user_id: string;
  email: string;
  display_name: string;
  source: 'rotation' | 'override';
};

export type ApiOnCallSlot = {
  id: string;
  team_id: string;
  user_id: string;
  start_at: string;
  end_at: string;
  source: 'rotation' | 'override';
};

export type ApiSchedule = {
  id: string;
  team_id: string;
  name: string;
  timezone: string;
  layers: Array<{
    handoff_weekday: number;
    handoff_time: string;
    participant_user_ids: string[];
  }>;
};

export type ApiOverride = {
  id: string;
  team_id: string;
  user_id: string;
  start_at: string;
  end_at: string;
};

export function monthRangeUTC(month: Date): { from: string; to: string } {
  const from = new Date(Date.UTC(month.getUTCFullYear(), month.getUTCMonth(), 1));
  const to = new Date(Date.UTC(month.getUTCFullYear(), month.getUTCMonth() + 1, 0, 23, 59, 59));
  return { from: from.toISOString(), to: to.toISOString() };
}

async function parseJson<T>(response: Response): Promise<T> {
  return (await response.json()) as T;
}

export async function fetchTeam(teamId: string): Promise<Team> {
  const response = await fetch(`/api/v1/teams/${teamId}`, { credentials: 'include' });
  if (!response.ok) {
    throw new Error('team fetch failed');
  }
  return parseJson<Team>(response);
}

export async function fetchTeamMembers(teamId: string): Promise<TeamMember[]> {
  const response = await fetch(`/api/v1/teams/${teamId}/members`, { credentials: 'include' });
  if (!response.ok) {
    throw new Error('members fetch failed');
  }
  const data = await parseJson<{ items: TeamMember[] }>(response);
  return data.items ?? [];
}

export async function fetchTeams(): Promise<Team[]> {
  const response = await fetch('/api/v1/teams', { credentials: 'include' });
  if (!response.ok) {
    throw new Error('teams fetch failed');
  }
  const data = await parseJson<{ items: Team[] }>(response);
  return data.items ?? [];
}

export async function fetchCurrentOnCall(teamId: string): Promise<ApiOnCallUser[]> {
  const response = await fetch(`/api/v1/teams/${teamId}/on-call/current`, { credentials: 'include' });
  if (!response.ok) {
    throw new Error('on-call fetch failed');
  }
  const data = await parseJson<{ items: ApiOnCallUser[] }>(response);
  return data.items ?? [];
}

export async function fetchOnCallCalendar(teamId: string, from: string, to: string): Promise<ApiOnCallSlot[]> {
  const params = new URLSearchParams({ from, to });
  const response = await fetch(`/api/v1/teams/${teamId}/on-call/calendar?${params}`, {
    credentials: 'include',
  });
  if (!response.ok) {
    throw new Error('calendar fetch failed');
  }
  const data = await parseJson<{ items: ApiOnCallSlot[] }>(response);
  return data.items ?? [];
}

export async function fetchTeamSchedules(teamId: string): Promise<ApiSchedule[]> {
  const response = await fetch(`/api/v1/teams/${teamId}/schedules`, { credentials: 'include' });
  if (!response.ok) {
    throw new Error('schedules fetch failed');
  }
  const data = await parseJson<{ items: ApiSchedule[] }>(response);
  return data.items ?? [];
}

export async function fetchTeamOverrides(teamId: string): Promise<ApiOverride[]> {
  const response = await fetch(`/api/v1/teams/${teamId}/overrides`, { credentials: 'include' });
  if (!response.ok) {
    throw new Error('overrides fetch failed');
  }
  const data = await parseJson<{ items: ApiOverride[] }>(response);
  return data.items ?? [];
}

export type ScheduleInput = {
  name: string;
  timezone: string;
  rotation: {
    handoff_weekday: number;
    handoff_time: string;
    participants: string[];
  };
};

export async function createSchedule(teamId: string, input: ScheduleInput): Promise<ApiSchedule> {
  const response = await fetch(`/api/v1/teams/${teamId}/schedules`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    const body = await parseJson<{ message?: string }>(response);
    throw new Error(body.message ?? 'schedule create failed');
  }
  return parseJson<ApiSchedule>(response);
}

export async function updateSchedule(teamId: string, scheduleId: string, input: ScheduleInput): Promise<ApiSchedule> {
  const response = await fetch(`/api/v1/teams/${teamId}/schedules/${scheduleId}`, {
    method: 'PATCH',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    const body = await parseJson<{ message?: string }>(response);
    throw new Error(body.message ?? 'schedule update failed');
  }
  return parseJson<ApiSchedule>(response);
}

export async function createOverride(
  teamId: string,
  input: { user_id: string; start_at: string; end_at: string },
): Promise<ApiOverride> {
  const response = await fetch(`/api/v1/teams/${teamId}/overrides`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    const body = await parseJson<{ message?: string }>(response);
    throw new Error(body.message ?? 'override create failed');
  }
  return parseJson<ApiOverride>(response);
}

export async function deleteOverride(teamId: string, overrideId: string): Promise<void> {
  const response = await fetch(`/api/v1/teams/${teamId}/overrides/${overrideId}`, {
    method: 'DELETE',
    credentials: 'include',
  });
  if (!response.ok) {
    const body = await parseJson<{ message?: string }>(response);
    throw new Error(body.message ?? 'override delete failed');
  }
}

export function memberNameMap(members: TeamMember[]): Map<string, string> {
  return new Map(members.map((member) => [member.user_id, member.display_name || member.email]));
}

export function mapApiToOnCallUsers(items: ApiOnCallUser[]): OnCallUser[] {
  return items.map((item) => ({
    userId: item.user_id,
    displayName: item.display_name,
    email: item.email,
    source: item.source,
  }));
}

export function mapApiCalendarSlots(
  items: ApiOnCallSlot[],
  nameByUserId: Map<string, string>,
): { slots: CalendarSlot[]; overrides: CalendarOverride[] } {
  const slots: CalendarSlot[] = [];
  const overrides: CalendarOverride[] = [];

  for (const item of items) {
    const displayName = nameByUserId.get(item.user_id) ?? item.user_id;
    if (item.source === 'override') {
      overrides.push({
        id: item.id,
        userId: item.user_id,
        displayName,
        startAt: item.start_at,
        endAt: item.end_at,
      });
    } else {
      slots.push({
        id: item.id,
        userId: item.user_id,
        displayName,
        startAt: item.start_at,
        endAt: item.end_at,
        source: 'rotation',
      });
    }
  }

  return { slots, overrides };
}
