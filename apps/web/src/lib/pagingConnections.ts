export type PagingProvider = 'slack' | 'express';

export type PagingConnection = {
  provider: PagingProvider;
  identity: string | null;
  available: boolean;
  unavailable_reason?: string;
};

export class PagingRequestError extends Error {
  constructor(public readonly code: string) {
    super(code);
  }
}

async function pagingRequest(path: string, method = 'GET'): Promise<Response> {
  const response = await fetch(`/api/v1/users/me/paging-connections${path}`, {
    method,
    credentials: 'include',
  });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({})) as { code?: string };
    throw new PagingRequestError(payload.code?.replace(/^PAGING_/, '').toLowerCase() ?? 'provider_unavailable');
  }
  return response;
}

export async function fetchPagingConnections(): Promise<PagingConnection[]> {
  const response = await pagingRequest('');
  return ((await response.json()) as { connections: PagingConnection[] }).connections;
}

export async function authorizePaging(provider: PagingProvider): Promise<string> {
  const response = await pagingRequest(`/${provider}/authorize`, 'POST');
  return ((await response.json()) as { authorization_url: string }).authorization_url;
}

export async function disconnectPaging(provider: PagingProvider): Promise<void> {
  await pagingRequest(`/${provider}`, 'DELETE');
}
