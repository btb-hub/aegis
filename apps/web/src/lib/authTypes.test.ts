import { afterEach, describe, expect, it, vi } from 'vitest';
import { createExpressLinkCode, fetchAuthProviders, patchAuthMe, parseAuthProviderIds } from './authTypes';

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

  it('parses configured provider ids and ignores unknown values', () => {
    expect(parseAuthProviderIds(['google', 'nope', 'slack'])).toEqual(['google', 'slack']);
    expect(parseAuthProviderIds(null)).toEqual([]);
  });

  it('returns no providers when the discovery request fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({}),
    }));
    await expect(fetchAuthProviders()).resolves.toEqual([]);
  });

  it('returns configured providers from discovery', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ providers: ['google'] }),
    }));
    await expect(fetchAuthProviders()).resolves.toEqual(['google']);
  });
});
