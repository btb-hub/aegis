import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import type { AuthUser } from '../lib/authTypes';
import { AccountPage } from './AccountPage';

const linkedUser: AuthUser = {
  id: 'user-1',
  email: 'alice@example.com',
  display_name: 'Alice Kim',
  role: 'admin',
  locale: 'en',
  provider: 'google',
  avatar_url: 'https://example.com/alice.png',
  slack_user_id: 'U123',
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
    (Story) => (
      <MemoryRouter>
        <Story />
      </MemoryRouter>
    ),
  ],
  parameters: {
    authUser: linkedUser,
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
