import type { Meta, StoryObj } from '@storybook/react';
import { Select } from './Select';

const meta: Meta<typeof Select> = {
  title: 'UI/Select',
  component: Select,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Select>;

export const Default: Story = {
  args: {
    label: 'Severity',
    value: '',
    options: [
      { value: '', label: 'Any' },
      { value: 'critical', label: 'Critical' },
      { value: 'warning', label: 'Warning' },
    ],
    onChange: () => undefined,
  },
};

export const Disabled: Story = {
  args: {
    ...Default.args,
    value: 'critical',
    disabled: true,
  },
};
