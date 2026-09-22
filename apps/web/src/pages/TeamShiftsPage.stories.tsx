import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { AppShell } from '../components/layout/AppShell';
import { AuthProvider } from '../context/AuthContext';
import type { CalendarOverride, CalendarSlot, OnCallUser } from '../lib/shiftsTypes';
import { TeamShiftsPage } from './TeamShiftsPage';

const june2026 = new Date('2026-06-01T00:00:00Z');

const onCallUsers: OnCallUser[] = [
  {
    userId: 'user-bob',
    displayName: 'Bob Chen',
    email: 'bob@example.com',
    source: 'rotation',
    contacts: {
      email: 'mailto:bob@example.com',
      slack: 'https://slack.com/app_redirect?channel=U456',
      express: 'https://xlnk.ms/open/profile/83fbf1c7-f14b-5176-bd32-ca15cf00d4b7',
    },
  },
];

const slots: CalendarSlot[] = [
  {
    id: 'slot-1',
    userId: 'user-alice',
    displayName: 'Alice Kim',
    startAt: '2026-06-02T09:00:00Z',
    endAt: '2026-06-09T09:00:00Z',
    source: 'rotation',
  },
  {
    id: 'slot-2',
    userId: 'user-bob',
    displayName: 'Bob Chen',
    startAt: '2026-06-09T09:00:00Z',
    endAt: '2026-06-16T09:00:00Z',
    source: 'rotation',
  },
];

const overrides: CalendarOverride[] = [
  {
    id: 'override-1',
    userId: 'user-carol',
    displayName: 'Carol Diaz',
    startAt: '2026-06-12T00:00:00Z',
    endAt: '2026-06-13T00:00:00Z',
  },
];

const meta: Meta<typeof TeamShiftsPage> = {
  title: 'Shifts/TeamShiftsPage',
  component: TeamShiftsPage,
  tags: ['autodocs'],
  args: {
    teamName: 'Platform',
    onCallUsers,
    slots,
    overrides,
    month: june2026,
  },
  decorators: [
    (Story) => (
      <MemoryRouter>
        <AuthProvider>
          <Story />
        </AuthProvider>
      </MemoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof TeamShiftsPage>;

export const English: Story = {
  globals: { locale: 'en' },
};

export const Russian: Story = {
  args: { teamName: 'Платформа' },
  globals: { locale: 'ru' },
};

export const NoOneOnCall: Story = {
  args: { onCallUsers: [] },
  globals: { locale: 'en' },
};

export const InAppShell: Story = {
  globals: { locale: 'en' },
  parameters: { layout: 'fullscreen' },
  render: (args) => (
    <AppShell>
      <TeamShiftsPage {...args} />
    </AppShell>
  ),
};

export const InAppShellRussian: Story = {
  args: { teamName: 'Платформа' },
  globals: { locale: 'ru' },
  parameters: { layout: 'fullscreen' },
  render: (args) => (
    <AppShell>
      <TeamShiftsPage {...args} />
    </AppShell>
  ),
};
