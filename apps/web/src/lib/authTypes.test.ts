import { afterEach, describe, expect, it, vi } from 'vitest';
import { createExpressLinkCode, patchAuthMe } from './authTypes';

describe('authTypes helpers', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('throws when profile patch fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ message: 'validation failed' }),
    }));

    await expect(patchAuthMe({ display_name: 'x' })).rejects.toThrow('validation failed');
  });

  it('throws generic error when patch response has no message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({}),
    }));

    await expect(patchAuthMe({ display_name: 'x' })).rejects.toThrow('profile update failed');
  });

  it('throws when express link code fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ message: 'code failed' }),
    }));

    await expect(createExpressLinkCode()).rejects.toThrow('code failed');
  });
});
