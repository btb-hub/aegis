import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../context/AuthContext';
import type { AuthUser } from '../lib/authTypes';
import { AccountPage } from './AccountPage';

const linkedUser: AuthUser = {
  id: 'user-1',
  email: 'alice@example.com',
  display_name: 'Alice Kim',
  role: 'admin',
  locale: 'en',
  provider: 'google',
  avatar_url: null,
  slack_user_id: 'U123',
  express_user_huid: '00000000-0000-4000-8000-000000000123',
  identities: [
    { provider: 'google', linked_at: '2026-01-01T00:00:00Z' },
    { provider: 'slack', linked_at: '2026-06-01T00:00:00Z' },
  ],
};

const meta: Meta<typeof AccountPage> = {
  title: 'Account/AccountPage',
  component: AccountPage,
  tags: ['autodocs'],
  decorators: [
    (Story, context) => (
      <MemoryRouter initialEntries={[context.parameters.accountPath ?? '/account']}>
        <AuthProvider><Story /></AuthProvider>
      </MemoryRouter>
    ),
  ],
  parameters: {
    authUser: linkedUser,
    pagingAvailable: true,
  },
  beforeEach: ({ parameters }) => {
    const originalFetch = window.fetch;
    const user = parameters.authUser as AuthUser;
    window.fetch = async (input, init) => {
      const path = String(input);
      let payload: unknown;
      if (path === '/auth/me') { payload = user; }
      else if (path === '/auth/providers') { payload = { providers: ['google', 'slack', 'express'] }; }
      else if (path === '/api/v1/users/me/paging-connections') {
        payload = { connections: [
          { provider: 'slack', identity: user.slack_user_id ?? null, available: parameters.pagingAvailable },
          { provider: 'express', identity: user.express_user_huid ?? null, available: parameters.pagingAvailable },
        ] };
      } else { return originalFetch(input, init); }
      return new Response(JSON.stringify(payload), { headers: { 'Content-Type': 'application/json' } });
    };
    return () => { window.fetch = originalFetch; };
  },
};

export default meta;
type Story = StoryObj<typeof AccountPage>;

export const LinkedProviders: Story = {
  globals: { locale: 'en' },
};

export const Russian: Story = {
  globals: { locale: 'ru' },
};

export const UnlinkedMessengers: Story = {
  parameters: { authUser: { ...linkedUser, slack_user_id: null, express_user_huid: null } },
};

export const SetupRequired: Story = {
  parameters: { pagingAvailable: false },
};

export const AuthorizationCancelled: Story = {
  parameters: { accountPath: '/account?paging_error=cancelled&paging_provider=express' },
};
