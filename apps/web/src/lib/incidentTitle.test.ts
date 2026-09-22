import { describe, expect, it } from 'vitest';
import { splitIncidentTitle } from './incidentTitle';

describe('splitIncidentTitle', () => {
  it('separates the readable summary from alert metadata', () => {
    expect(
      splitIncidentTitle(
        'HelmRelease is not ready > 10 min. Where: alertname: FluxHelmReleaseNotReady namespace: flux-system',
      ),
    ).toEqual({
      summary: 'HelmRelease is not ready > 10 min.',
      context: 'Where: alertname: FluxHelmReleaseNotReady namespace: flux-system',
    });
  });

  it('normalizes whitespace when no context marker exists', () => {
    expect(splitIncidentTitle('  CPU   is\n high  ')).toEqual({ summary: 'CPU is high' });
  });
});
