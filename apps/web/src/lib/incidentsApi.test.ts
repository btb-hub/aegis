import { describe, expect, it, vi } from 'vitest';
import {
  acknowledgeIncident,
  bounceIncident,
  fetchHandoffTargets,
  fetchIncidents,
  handoffIncident,
  mapApiIncident,
  mapApiTimelineEvent,
  resolveIncident,
} from './incidentsApi';

describe('incidentsApi', () => {
  it('maps timeline payload values to strings', () => {
    const event = mapApiTimelineEvent({
      id: 'e1',
      kind: 'handoff',
      payload: { to_team_id: 'team-2', note: 'needs help', count: 2 },
      created_at: '2026-06-26T10:00:00Z',
    });

    expect(event).toEqual({
      id: 'e1',
      kind: 'handoff',
      payload: { to_team_id: 'team-2', note: 'needs help', count: '2' },
      createdAt: '2026-06-26T10:00:00Z',
    });
  });

  it('maps incident detail with alerts and timeline', () => {
    const incident = mapApiIncident(
      {
        id: '11111111-1111-1111-1111-111111111111',
        team_id: 'team-1',
        status: 'open',
        severity: 'critical',
        title: 'CPU high',
        fingerprint: 'fp-1',
        jira_issue_key: 'OPS-1',
        created_at: '2026-06-26T10:00:00Z',
      },
      [{ id: 'a1', severity: 'critical', title: 'CPU high', status: 'firing' }],
      [{ id: 'e1', kind: 'created', payload: {}, createdAt: '2026-06-26T10:00:00Z' }],
    );

    expect(incident).toMatchObject({
      id: '11111111-1111-1111-1111-111111111111',
      teamId: 'team-1',
      jiraIssueKey: 'OPS-1',
      alerts: [{ id: 'a1', severity: 'critical', title: 'CPU high', status: 'firing' }],
      timeline: [{ id: 'e1', kind: 'created', payload: {}, createdAt: '2026-06-26T10:00:00Z' }],
    });
  });

  it('loads incidents and handoff targets from the API', async () => {
    vi.stubGlobal('fetch', vi.fn());
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        jsonResponse({
          items: [{ id: 'i1', team_id: 'team-1', status: 'open', severity: 'critical', title: 'CPU', fingerprint: 'fp', created_at: '' }],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [{ id: 'team-2', name: 'L3', support_tier: 'l3' }] }));

    const incidents = await fetchIncidents('open');
    expect(incidents).toHaveLength(1);
    expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/v1/incidents?status=open', expect.any(Object));

    const targets = await fetchHandoffTargets('team-1');
    expect(targets[0].name).toBe('L3');
    vi.unstubAllGlobals();
  });

  it('posts incident actions', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 204, json: async () => ({}) }));

    await acknowledgeIncident('i1');
    await resolveIncident('i1');
    await handoffIncident('i1', 'team-2', 'needs help');
    await bounceIncident('i1', 'wrong team');

    expect(vi.mocked(fetch)).toHaveBeenCalledWith('/api/v1/incidents/i1/handoff', expect.objectContaining({ method: 'POST' }));
    vi.unstubAllGlobals();
  });
});

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}
