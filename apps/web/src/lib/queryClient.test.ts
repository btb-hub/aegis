import { describe, expect, it, vi } from 'vitest';
import { ApiError, apiFetch } from './apiClient';
import { createAppQueryClient, incidentQueryKeys, queryKeys } from './queryClient';

describe('createAppQueryClient', () => {
  it('uses conservative fetch defaults', () => {
    const client = createAppQueryClient();
    const defaults = client.getDefaultOptions().queries;

    expect(defaults?.retry).toBe(1);
    expect(defaults?.refetchOnWindowFocus).toBe(false);
    expect(defaults?.staleTime).toBe(15_000);
  });
});

describe('incidentQueryKeys', () => {
  it('nests list and detail under incidents', () => {
    expect(incidentQueryKeys.list('open')[0]).toBe(incidentQueryKeys.all[0]);
    expect(incidentQueryKeys.detail('id-1')[0]).toBe(incidentQueryKeys.all[0]);
    expect(queryKeys.incidents).toBe(incidentQueryKeys);
  });
});

describe('apiFetch', () => {
  it('returns JSON on success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ id: '1' }),
      }),
    );

    await expect(apiFetch<{ id: string }>('/api/v1/x')).resolves.toEqual({ id: '1' });
    vi.unstubAllGlobals();
  });

  it('returns undefined for empty success bodies', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 204,
        json: async () => {
          throw new Error('empty');
        },
      }),
    );

    await expect(apiFetch('/api/v1/x', { method: 'DELETE' })).resolves.toBeUndefined();
    vi.unstubAllGlobals();
  });

  it('throws ApiError with status and body on failure', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        json: async () => ({ message: 'blocked', code: 'conflict' }),
      }),
    );

    await expect(apiFetch('/api/v1/x')).rejects.toMatchObject({
      name: 'ApiError',
      status: 409,
      message: 'blocked',
      code: 'conflict',
    });
    expect(new ApiError('blocked', 409) instanceof ApiError).toBe(true);
    vi.unstubAllGlobals();
  });

  it('throws a fallback ApiError when the error body is not JSON', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => {
          throw new Error('no json');
        },
      }),
    );

    await expect(apiFetch('/api/v1/x')).rejects.toMatchObject({
      name: 'ApiError',
      status: 500,
      message: 'request failed: 500',
    });
    vi.unstubAllGlobals();
  });
});
