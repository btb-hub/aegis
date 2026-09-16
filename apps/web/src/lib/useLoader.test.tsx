import { renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { describe, expect, it } from 'vitest';
import { TestQueryProvider } from '../test/renderWithQuery';
import { useLoader } from './useLoader';

function wrapper({ children }: { children: ReactNode }) {
  return <TestQueryProvider>{children}</TestQueryProvider>;
}

describe('useLoader', () => {
  it('exposes data after the query resolves', async () => {
    const { result } = renderHook(() => useLoader(['loader', 'ok'], async () => ({ id: '1' })), {
      wrapper,
    });

    expect(result.current.loading).toBe(true);

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });
    expect(result.current.data).toEqual({ id: '1' });
    expect(result.current.isError).toBe(false);
  });

  it('surfaces query errors', async () => {
    const { result } = renderHook(
      () =>
        useLoader(['loader', 'err'], async () => {
          throw new Error('nope');
        }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });
    expect(result.current.error?.message).toBe('nope');
    expect(result.current.loading).toBe(false);
  });

  it('skips the request when disabled', async () => {
    const { result } = renderHook(
      () => useLoader(['loader', 'off'], async () => ({ id: '1' }), { enabled: false }),
      { wrapper },
    );

    expect(result.current.loading).toBe(false);
    expect(result.current.data).toBeUndefined();
    await waitFor(() => {
      expect(result.current.query.fetchStatus).toBe('idle');
    });
    expect(result.current.data).toBeUndefined();
    expect(result.current.loading).toBe(false);
  });
});
