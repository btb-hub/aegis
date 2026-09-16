import { apiFetch, UsersApiError } from './apiClient';

export type UserRole = 'admin' | 'member' | 'viewer';

export type ListedUser = {
  id: string;
  email: string;
  display_name: string;
  role: UserRole;
  role_pinned?: boolean;
};

export { UsersApiError };

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
