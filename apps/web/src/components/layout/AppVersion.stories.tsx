import type { Meta, StoryObj } from '@storybook/react';
import { AppVersion } from './AppVersion';

const meta: Meta<typeof AppVersion> = {
  title: 'Layout/AppVersion',
  component: AppVersion,
  tags: ['autodocs'],
  parameters: { layout: 'fullscreen' },
};

export default meta;
type Story = StoryObj<typeof AppVersion>;

export const Default: Story = {};
