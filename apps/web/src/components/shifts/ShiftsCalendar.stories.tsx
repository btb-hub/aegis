import type { Meta, StoryObj } from '@storybook/react';
import type { CalendarOverride, CalendarSlot } from '../../lib/shiftsTypes';
import { ShiftsCalendar } from './ShiftsCalendar';

const june2026 = new Date('2026-06-01T00:00:00Z');

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
