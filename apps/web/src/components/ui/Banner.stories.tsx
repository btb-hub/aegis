import type { Meta, StoryObj } from '@storybook/react';
import { Banner } from './Banner';

const meta: Meta<typeof Banner> = {
  title: 'UI/Banner',
  component: Banner,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Banner>;

export const Info: Story = {
  args: { children: 'Filters updated.', variant: 'info' },
};

export const Warning: Story = {
  args: { children: 'Could not load alerts.', variant: 'warning' },
};

export const Error: Story = {
  args: { children: 'Export failed.', variant: 'error' },
};
