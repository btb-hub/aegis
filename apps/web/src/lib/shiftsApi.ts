import { apiFetch } from './apiClient';
import type { ContactLinks } from './contactTypes';
import type { CalendarOverride, CalendarSlot, OnCallUser } from './shiftsTypes';
import type { Team, TeamMember } from './teamTypes';

export type ApiOnCallUser = {
  user_id: string;
  email: string;
  display_name: string;
  source: 'rotation' | 'override';
  contacts?: ContactLinks;
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

export async function fetchTeam(teamId: string): Promise<Team> {
  return apiFetch<Team>(`/api/v1/teams/${teamId}`);
}

export async function fetchTeamMembers(teamId: string): Promise<TeamMember[]> {
  const data = await apiFetch<{ items: TeamMember[] }>(`/api/v1/teams/${teamId}/members`);
  return data.items ?? [];
}

export async function fetchTeams(workspaceId?: string): Promise<Team[]> {
  const query =
    workspaceId && workspaceId !== 'all'
      ? `?workspace_id=${encodeURIComponent(workspaceId)}`
      : '';
  const data = await apiFetch<{ items: Team[] }>(`/api/v1/teams${query}`);
  return data.items ?? [];
}

export async function createTeam(payload: Record<string, string>): Promise<Team> {
  return apiFetch<Team>('/api/v1/teams', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function updateTeam(teamId: string, payload: Record<string, string>): Promise<Team> {
  return apiFetch<Team>(`/api/v1/teams/${teamId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function deleteTeam(teamId: string): Promise<void> {
  await apiFetch(`/api/v1/teams/${teamId}`, { method: 'DELETE' });
}

export async function fetchCurrentOnCall(teamId: string): Promise<ApiOnCallUser[]> {
  const data = await apiFetch<{ items: ApiOnCallUser[] }>(`/api/v1/teams/${teamId}/on-call/current`);
  return data.items ?? [];
}

export async function fetchOnCallCalendar(teamId: string, from: string, to: string): Promise<ApiOnCallSlot[]> {
  const params = new URLSearchParams({ from, to });
  const data = await apiFetch<{ items: ApiOnCallSlot[] }>(
    `/api/v1/teams/${teamId}/on-call/calendar?${params}`,
  );
  return data.items ?? [];
}

export async function fetchTeamSchedules(teamId: string): Promise<ApiSchedule[]> {
  const data = await apiFetch<{ items: ApiSchedule[] }>(`/api/v1/teams/${teamId}/schedules`);
  return data.items ?? [];
}

export async function fetchTeamOverrides(teamId: string): Promise<ApiOverride[]> {
  const data = await apiFetch<{ items: ApiOverride[] }>(`/api/v1/teams/${teamId}/overrides`);
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
  return apiFetch<ApiSchedule>(`/api/v1/teams/${teamId}/schedules`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  });
}

export async function updateSchedule(teamId: string, scheduleId: string, input: ScheduleInput): Promise<ApiSchedule> {
  return apiFetch<ApiSchedule>(`/api/v1/teams/${teamId}/schedules/${scheduleId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  });
}

export async function createOverride(
  teamId: string,
  input: { user_id: string; start_at: string; end_at: string },
): Promise<ApiOverride> {
  return apiFetch<ApiOverride>(`/api/v1/teams/${teamId}/overrides`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  });
}

export async function deleteOverride(teamId: string, overrideId: string): Promise<void> {
  await apiFetch(`/api/v1/teams/${teamId}/overrides/${overrideId}`, { method: 'DELETE' });
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
    contacts: item.contacts,
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
