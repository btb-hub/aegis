import type { Meta, StoryObj } from '@storybook/react';
import type { OnCallUser } from '../../lib/shiftsTypes';
import { OnCallBanner } from './OnCallBanner';

const onCallUser: OnCallUser = {
  userId: 'user-bob',
  displayName: 'Bob Chen',
  email: 'bob@example.com',
  source: 'rotation',
};

const overrideUser: OnCallUser = {
  userId: 'user-carol',
  displayName: 'Carol Diaz',
  email: 'carol@example.com',
  source: 'override',
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
