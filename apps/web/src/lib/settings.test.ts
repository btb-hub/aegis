import { afterEach, expect, it, vi } from 'vitest';
import { settingsRequest } from './settings';
afterEach(() => vi.unstubAllGlobals());
it('returns only localized error keys for network, invalid JSON, and API failures', async () => {
  for (const response of [Promise.reject(new Error('private response')), Promise.resolve({ ok: true, json: () => Promise.reject(new Error('bad JSON')) }), Promise.resolve({ ok: false, json: () => Promise.reject(new Error('HTML')) }), Promise.resolve({ ok: false, json: async () => ({ code: 'VALIDATION' }) })]) {
    vi.stubGlobal('fetch', vi.fn(() => response));
    await expect(settingsRequest('/api/v1/settings')).rejects.toThrow('settings.request_failed');
  }
});
