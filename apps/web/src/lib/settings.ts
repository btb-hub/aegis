export type Settings = {
  values: Record<string, string>;
  secret_present: Record<string, boolean>;
  enabled: Record<string, boolean>;
  revision: number;
  bootstrap_closed: boolean;
};
export type ProviderDraft = {
  provider: string;
  values: Record<string, string>;
  secret_present: Record<string, boolean>;
  revision: number;
  tested: boolean;
  expected_email: string;
};

export async function settingsRequest<T>(path: string, method = 'GET', body?: unknown): Promise<T> {
  try {
  const response = await fetch(path, {
    method, credentials: 'include',
    ...(body === undefined ? {} : { headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({})) as { code?: string };
    throw new Error(error.code === 'CONFLICT' ? 'settings.conflict' : 'settings.request_failed');
  }
  return await response.json() as T;
  } catch (error) {
    throw new Error(error instanceof Error && error.message === 'settings.conflict' ? error.message : 'settings.request_failed');
  }
}
