import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  addEscalationPath,
  createRoutingRule,
  createWorkspace,
  deleteEscalationPath,
  deleteRoutingRule,
  fetchEscalationPaths,
  fetchIncomingPaths,
  fetchOutgoingPaths,
  fetchRoutingRules,
  fetchWorkspace,
  fetchWorkspaces,
  updateRoutingRule,
} from './workspacesApi';

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}

describe('workspacesApi', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('fetchWorkspaces returns items array', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({
        items: [{ id: 'ws-1', name: 'Default', slug: 'default', description: '', created_at: '', updated_at: '' }],
      }),
    );

    const items = await fetchWorkspaces();
    expect(items).toHaveLength(1);
    expect(items[0].name).toBe('Default');
  });

  it('fetchWorkspace loads a single workspace', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({ id: 'ws-1', name: 'Platform', slug: 'platform', description: 'Core' }),
    );

    const workspace = await fetchWorkspace('ws-1');
    expect(workspace.slug).toBe('platform');
  });

  it('fetchRoutingRules returns parsed rules', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({
        items: [
          {
            id: 'rule-1',
            team_id: 'team-1',
            match_labels: { team: 'platform' },
            priority: 10,
            created_at: '',
            updated_at: '',
          },
        ],
      }),
    );

    const rules = await fetchRoutingRules();
    expect(rules[0].match_labels.team).toBe('platform');
  });

  it('createRoutingRule posts payload', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({
        id: 'rule-2',
        team_id: 'team-1',
        match_labels: { service: 'payments' },
        priority: 50,
        created_at: '',
        updated_at: '',
      }),
    );

    const rule = await createRoutingRule({
      team_id: 'team-1',
      match_labels: { service: 'payments' },
      priority: 50,
    });
    expect(rule.priority).toBe(50);
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      '/api/v1/routing-rules',
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('updateRoutingRule patches payload', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({
        id: 'rule-1',
        team_id: 'team-1',
        match_labels: { team: 'core' },
        priority: 20,
        created_at: '',
        updated_at: '',
      }),
    );

    const rule = await updateRoutingRule('rule-1', {
      team_id: 'team-1',
      match_labels: { team: 'core' },
      priority: 20,
    });
    expect(rule.match_labels.team).toBe('core');
  });

  it('deleteRoutingRule sends delete request', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({}, 204));

    await deleteRoutingRule('rule-1');
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      '/api/v1/routing-rules/rule-1',
      expect.objectContaining({ method: 'DELETE' }),
    );
  });

  it('throws when response is not ok', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ message: 'bad request' }, 400));

    await expect(fetchWorkspace('missing')).rejects.toThrow('request failed: 400');
  });

  it('covers workspace and escalation path helpers', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ id: 'ws-1', name: 'Platform', slug: 'platform', description: '' }))
      .mockResolvedValueOnce(jsonResponse({ items: [{ id: 'path-1', from_team_id: 't1', to_team_id: 't2', workspace_id: 'ws-1', cross_workspace: false, created_at: '' }] }))
      .mockResolvedValueOnce(jsonResponse({ id: 'path-2', from_team_id: 't1', to_team_id: 't2', workspace_id: 'ws-1', cross_workspace: false, created_at: '' }, 201))
      .mockResolvedValueOnce(jsonResponse({}, 204))
      .mockResolvedValueOnce(jsonResponse({ items: [{ id: 'path-out', from_team_id: 't1', to_team_id: 't2', workspace_id: 'ws-1', cross_workspace: false, created_at: '' }] }))
      .mockResolvedValueOnce(jsonResponse({ items: [{ id: 'path-in', from_team_id: 't1', to_team_id: 't2', workspace_id: 'ws-1', cross_workspace: false, created_at: '' }] }));

    const workspace = await createWorkspace({ name: 'Platform', description: 'Core' });
    expect(workspace.name).toBe('Platform');

    const paths = await fetchEscalationPaths('ws-1');
    expect(paths).toHaveLength(1);

    const createdPath = await addEscalationPath('ws-1', { from_team_id: 't1', to_team_id: 't2' });
    expect(createdPath.id).toBe('path-2');

    await deleteEscalationPath('path-1');

    const outgoing = await fetchOutgoingPaths('t1');
    expect(outgoing[0].id).toBe('path-out');

    const incoming = await fetchIncomingPaths('t2');
    expect(incoming[0].id).toBe('path-in');
  });
});
