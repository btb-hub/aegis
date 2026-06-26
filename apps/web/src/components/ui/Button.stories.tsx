import type { Meta, StoryObj } from '@storybook/react';
import { Button } from './Button';

const meta: Meta<typeof Button> = {
  title: 'UI/Button',
  component: Button,
  tags: ['autodocs'],
  argTypes: {
    variant: { control: 'select', options: ['primary', 'secondary', 'ghost'] },
    onClick: { action: 'clicked' },
  },
};

export default meta;
type Story = StoryObj<typeof Button>;

export const Primary: Story = {
  args: { children: 'Primary action', variant: 'primary' },
};

export const PrimaryHover: Story = {
  args: { children: 'Primary action', variant: 'primary' },
  parameters: { pseudo: { hover: true } },
};

export const PrimaryDisabled: Story = {
  args: { children: 'Primary action', variant: 'primary', disabled: true },
};

export const Secondary: Story = {
  args: { children: 'Secondary action', variant: 'secondary' },
};

export const SecondaryHover: Story = {
  args: { children: 'Secondary action', variant: 'secondary' },
  parameters: { pseudo: { hover: true } },
};

export const SecondaryDisabled: Story = {
  args: { children: 'Secondary action', variant: 'secondary', disabled: true },
};

export const Ghost: Story = {
  args: { children: 'Ghost action', variant: 'ghost' },
};

export const GhostHover: Story = {
  args: { children: 'Ghost action', variant: 'ghost' },
  parameters: { pseudo: { hover: true } },
};

export const GhostDisabled: Story = {
  args: { children: 'Ghost action', variant: 'ghost', disabled: true },
};
