export type UserIdentity = {
  provider: string;
  provider_sub?: string;
  linked_at: string;
};

export type AuthUser = {
  id: string;
  email: string;
  display_name: string;
  role: string;
  locale: string;
  provider: string;
  avatar_url?: string | null;
  slack_user_id?: string | null;
  express_user_huid?: string | null;
  identities?: UserIdentity[];
};

export type AuthProviderId = 'google' | 'slack' | 'express';

export const AUTH_PROVIDERS: AuthProviderId[] = ['google', 'slack', 'express'];

export async function patchAuthMe(body: { locale?: string; display_name?: string }): Promise<AuthUser> {
  const response = await fetch('/auth/me', {
    method: 'PATCH',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const payload = (await response.json()) as { message?: string };
    throw new Error(payload.message ?? 'profile update failed');
  }
  return (await response.json()) as AuthUser;
}

export async function createExpressLinkCode(): Promise<{ code: string; command: string }> {
  const response = await fetch('/api/v1/users/me/express-link-code', {
    method: 'POST',
    credentials: 'include',
  });
  if (!response.ok) {
    const payload = (await response.json()) as { message?: string };
    throw new Error(payload.message ?? 'link code failed');
  }
  return (await response.json()) as { code: string; command: string };
}
