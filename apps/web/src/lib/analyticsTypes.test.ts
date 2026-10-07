import { describe, expect, it } from 'vitest';
import { formatDuration } from './analyticsTypes';

describe('analyticsTypes', () => {
  it('formats short durations', () => {
    expect(formatDuration(45)).toBe('45s');
    expect(formatDuration(120)).toBe('2m');
    expect(formatDuration(7200)).toBe('2.0h');
  });
});
