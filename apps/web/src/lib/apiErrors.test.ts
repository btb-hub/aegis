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
    expect(resolveApiErrorMessage(t, { message: 'something else' }, 'fallback')).toBe('something else');
  });

  it('maps handoff no on-call with team name', () => {
    expect(
      resolveApiErrorMessage(
        (key, options) => `${key}:${JSON.stringify(options)}`,
        {
          message: 'target team has no one on call',
          details: { team_name: 'Helpdesk' },
        },
        'fallback',
      ),
    ).toBe('errors.handoff_no_on_call_named:{"team":"Helpdesk"}');
  });
});
