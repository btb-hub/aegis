import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  addEscalationPath,
  createRoutingRule,
  createWorkspace,
  deleteEscalationPath,
  deleteRoutingRule,
  assignTeamsToWorkspace,
  deleteWorkspace,
  updateWorkspace,
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
        items: [
          {
            id: 'ws-1',
            name: 'Default',
            slug: 'default',
            description: '',
            team_count: 3,
            routing_rule_count: 2,
            created_at: '',
            updated_at: '',
          },
        ],
      }),
    );

    const items = await fetchWorkspaces();
    expect(items).toHaveLength(1);
    expect(items[0].name).toBe('Default');
    expect(items[0].team_count).toBe(3);
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

    await expect(fetchWorkspace('missing')).rejects.toThrow('bad request');
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

  it('assignTeamsToWorkspace posts team_ids', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({
        items: [{ id: 'team-1', workspace_id: 'ws-1', name: 'Platform', description: '', created_at: '', updated_at: '' }],
      }),
    );

    const teams = await assignTeamsToWorkspace('ws-1', ['team-1']);
    expect(teams).toHaveLength(1);
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      '/api/v1/workspaces/ws-1/teams',
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('updateWorkspace and deleteWorkspace call patch and delete', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        jsonResponse({ id: 'ws-1', name: 'Platform Ops', slug: 'platform', description: 'Updated' }),
      )
      .mockResolvedValueOnce(jsonResponse({}, 204));

    const updated = await updateWorkspace('ws-1', { name: 'Platform Ops', description: 'Updated' });
    expect(updated.name).toBe('Platform Ops');
    await deleteWorkspace('ws-1');
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      '/api/v1/workspaces/ws-1',
      expect.objectContaining({ method: 'DELETE' }),
    );
  });
});
