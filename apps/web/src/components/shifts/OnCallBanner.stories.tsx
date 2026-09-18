import type { Meta, StoryObj } from '@storybook/react';
import type { OnCallUser } from '../../lib/shiftsTypes';
import { OnCallBanner } from './OnCallBanner';

const onCallUser: OnCallUser = {
  userId: 'user-bob',
  displayName: 'Bob Chen',
  email: 'bob@example.com',
  source: 'rotation',
  contacts: {
    email: 'mailto:bob@example.com',
    slack: 'https://slack.com/app_redirect?channel=U456',
    express: 'https://xlnk.ms/open/profile/83fbf1c7-f14b-5176-bd32-ca15cf00d4b7',
  },
};

const overrideUser: OnCallUser = {
  userId: 'user-carol',
  displayName: 'Carol Diaz',
  email: 'carol@example.com',
  source: 'override',
  contacts: { email: 'mailto:carol@example.com' },
};

const meta: Meta<typeof OnCallBanner> = {
  title: 'Shifts/OnCallBanner',
  component: OnCallBanner,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof OnCallBanner>;

export const OnCallNow: Story = {
  args: { users: [onCallUser] },
  globals: { locale: 'en' },
};

export const OnCallNowRussian: Story = {
  args: { users: [onCallUser] },
  globals: { locale: 'ru' },
};

export const OnCallOverride: Story = {
  args: { users: [overrideUser] },
  globals: { locale: 'en' },
};

export const MultipleOnCall: Story = {
  args: {
    users: [
      onCallUser,
      {
        userId: 'user-alice',
        displayName: 'Alice Kim',
        email: 'alice@example.com',
        source: 'rotation',
      },
    ],
  },
  globals: { locale: 'en' },
};

export const Empty: Story = {
  args: { users: [] },
  globals: { locale: 'en' },
};

export const EmptyRussian: Story = {
  args: { users: [] },
  globals: { locale: 'ru' },
};
