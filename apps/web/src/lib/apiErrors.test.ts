import { describe, expect, it } from 'vitest';
import { resolveApiErrorMessage } from './apiErrors';

const t = (key: string) => `localized:${key}`;

describe('resolveApiErrorMessage', () => {
  it('maps known API messages to locale keys', () => {
    expect(
      resolveApiErrorMessage(t, { message: 'team name is required' }, 'fallback'),
    ).toBe('localized:errors.team_name_required');
  });

  it('returns fallback for unknown messages', () => {
    expect(resolveApiErrorMessage(t, { message: 'something else' }, 'fallback')).toBe('fallback');
  });
});
