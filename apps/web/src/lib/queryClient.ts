import { QueryClient } from '@tanstack/react-query';

export const incidentQueryKeys = {
  all: ['incidents'] as const,
  list: (status: string) => ['incidents', 'list', status] as const,
  detail: (id: string) => ['incidents', 'detail', id] as const,
};

export const queryKeys = {
  incidents: incidentQueryKeys,
  alerts: {
    all: ['alerts'] as const,
    list: (params: string) => ['alerts', 'list', params] as const,
    savedViews: ['alerts', 'saved-views'] as const,
  },
  teams: {
    all: ['teams'] as const,
    list: (workspaceId: string) => ['teams', 'list', workspaceId] as const,
    detail: (id: string) => ['teams', 'detail', id] as const,
    members: (id: string) => ['teams', 'members', id] as const,
  },
  users: {
    all: ['users'] as const,
    list: (q = '') => ['users', 'list', q] as const,
  },
  workspaces: {
    all: ['workspaces'] as const,
    list: ['workspaces', 'list'] as const,
    detail: (id: string) => ['workspaces', 'detail', id] as const,
  },
  dashboard: {
    overview: (compare: boolean) => ['dashboard', 'overview', compare] as const,
  },
  integrations: {
    all: ['integrations'] as const,
    list: ['integrations', 'list'] as const,
  },
  shifts: {
    landing: ['shifts', 'landing'] as const,
    team: (id: string) => ['shifts', 'team', id] as const,
  },
  setup: {
    health: ['setup', 'health'] as const,
    wizardWorkspaces: ['setup', 'workspaces'] as const,
  },
};

export function createAppQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: 1,
        refetchOnWindowFocus: false,
        staleTime: 15_000,
      },
    },
  });
}
