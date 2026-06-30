export type AuthUser = {
  id: string;
  email: string;
  display_name: string;
  role: string;
  locale: string;
  provider: string;
};

export type AuthProviderId = 'google' | 'slack' | 'express';

export const AUTH_PROVIDERS: AuthProviderId[] = ['google', 'slack', 'express'];
