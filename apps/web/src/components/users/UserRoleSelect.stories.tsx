import type { Meta, StoryObj } from '@storybook/react';
import { UserRoleSelect } from './UserRoleSelect';

const meta: Meta<typeof UserRoleSelect> = {
  title: 'Users/UserRoleSelect',
  component: UserRoleSelect,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof UserRoleSelect>;

export const English: Story = {
  globals: { locale: 'en' },
  args: {
    id: 'story-role-en',
    label: 'Role',
    hideLabel: true,
    value: 'member',
    onChange: () => undefined,
  },
};

export const Russian: Story = {
  globals: { locale: 'ru' },
  args: {
    id: 'story-role-ru',
    label: 'Роль',
    hideLabel: true,
    value: 'member',
    onChange: () => undefined,
  },
};

export const Pinned: Story = {
  args: {
    id: 'story-role-pinned',
    label: 'Role',
    hideLabel: true,
    value: 'admin',
    pinned: true,
    onChange: () => undefined,
  },
};
