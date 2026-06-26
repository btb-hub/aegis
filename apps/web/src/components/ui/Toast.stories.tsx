import type { Meta, StoryObj } from '@storybook/react';
import { Toast } from './Toast';

const meta: Meta<typeof Toast> = {
  title: 'UI/Toast',
  component: Toast,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;
type Story = StoryObj<typeof Toast>;

export const Default: Story = {
  args: { message: 'Settings saved' },
};

export const Success: Story = {
  args: { message: 'Acknowledged', variant: 'success' },
};
