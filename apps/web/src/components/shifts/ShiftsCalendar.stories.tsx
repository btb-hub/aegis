import type { Meta, StoryObj } from '@storybook/react';
import type { CalendarOverride, CalendarSlot } from '../../lib/shiftsTypes';
import { ShiftsCalendar } from './ShiftsCalendar';

const june2026 = new Date('2026-06-12T00:00:00Z');

const demoSlots: CalendarSlot[] = [
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
  {
    id: 'slot-3',
    userId: 'user-alice',
    displayName: 'Alice Kim',
    startAt: '2026-06-16T09:00:00Z',
    endAt: '2026-06-23T09:00:00Z',
    source: 'rotation',
  },
];

const demoOverrides: CalendarOverride[] = [
  {
    id: 'override-1',
    userId: 'user-carol',
    displayName: 'Carol Diaz',
    startAt: '2026-06-12T00:00:00Z',
    endAt: '2026-06-13T00:00:00Z',
  },
];

const meta: Meta<typeof ShiftsCalendar> = {
  title: 'Shifts/ShiftsCalendar',
  component: ShiftsCalendar,
  tags: ['autodocs'],
  args: {
    month: june2026,
    teamName: 'Platform',
    slots: demoSlots,
    overrides: demoOverrides,
  },
};

export default meta;
type Story = StoryObj<typeof ShiftsCalendar>;

export const WithRotationsAndOverrides: Story = {
  globals: { locale: 'en' },
};

export const WithRotationsAndOverridesRussian: Story = {
  globals: { locale: 'ru' },
};

export const RotationsOnly: Story = {
  args: { overrides: [] },
  globals: { locale: 'en' },
};

export const EmptyMonth: Story = {
  args: { slots: [], overrides: [] },
  globals: { locale: 'en' },
};

export const DesignReference: Story = {
  args: {
    month: new Date('2026-06-24T00:00:00Z'),
    lanes: [
      {
        id: 'payments', name: 'Payments',
        slots: [{ id: 'payments-jamie', userId: 'jamie', displayName: 'Jamie Lee', startAt: '2026-06-22T00:00:00Z', endAt: '2026-06-29T00:00:00Z', source: 'rotation' }],
        overrides: [{ id: 'payments-kit', userId: 'kit', displayName: 'Kit T', startAt: '2026-06-24T00:00:00Z', endAt: '2026-06-25T00:00:00Z' }],
      },
      {
        id: 'search', name: 'Search',
        slots: [{ id: 'search-maya', userId: 'maya', displayName: 'Maya R', startAt: '2026-06-22T00:00:00Z', endAt: '2026-06-29T00:00:00Z', source: 'rotation' }], overrides: [],
      },
      {
        id: 'platform', name: 'Platform',
        slots: [
          { id: 'platform-jia', userId: 'jia', displayName: 'Jia X', startAt: '2026-06-22T00:00:00Z', endAt: '2026-06-25T00:00:00Z', source: 'rotation' },
          { id: 'platform-sam', userId: 'sam', displayName: 'Sam C', startAt: '2026-06-25T00:00:00Z', endAt: '2026-06-29T00:00:00Z', source: 'rotation' },
        ], overrides: [],
      },
      {
        id: 'platform-l3', name: 'Platform L3',
        slots: [{ id: 'l3-priya', userId: 'priya', displayName: 'Priya V', startAt: '2026-06-22T00:00:00Z', endAt: '2026-06-29T00:00:00Z', source: 'rotation' }], overrides: [],
      },
    ],
  },
  globals: { locale: 'en' },
};
