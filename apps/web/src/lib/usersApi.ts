export type UserRole = 'admin' | 'member' | 'viewer';

export type ListedUser = {
  id: string;
  email: string;
  display_name: string;
  role: UserRole;
  role_pinned?: boolean;
};

export class UsersApiError extends Error {
  status: number;
  code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, { credentials: 'include', ...init });
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as {
      message?: string;
      code?: string;
    };
    throw new UsersApiError(body.message ?? `request failed: ${response.status}`, response.status, body.code);
  }
  return (await response.json()) as T;
}

export async function fetchUsers(q = ''): Promise<{ items: ListedUser[] }> {
  const query = q.trim() ? `?q=${encodeURIComponent(q.trim())}` : '';
  return apiFetch<{ items: ListedUser[] }>(`/api/v1/users${query}`);
}

export async function patchUserRole(id: string, role: string): Promise<ListedUser> {
  return apiFetch<ListedUser>(`/api/v1/users/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ role }),
  });
}
