import { describe, expect, it } from 'vitest';
import { severityLabelKey, severityToTag } from './severityTag';

describe('severityToTag', () => {
  it('maps API severity strings to P1-P4', () => {
    expect(severityToTag('critical')).toBe('P1');
    expect(severityToTag('warning')).toBe('P2');
    expect(severityToTag('moderate')).toBe('P3');
    expect(severityToTag('info')).toBe('P4');
  });

  it('returns neutral for unknown severities', () => {
    expect(severityToTag('unknown')).toBe('neutral');
  });

  it('builds locale keys from severity values', () => {
    expect(severityLabelKey('Critical')).toBe('incidents.severity.critical');
  });
});
