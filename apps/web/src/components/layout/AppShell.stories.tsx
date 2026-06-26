import type { Meta, StoryObj } from '@storybook/react';
import { SeverityTag } from '../ui/SeverityTag';
import { Button } from '../ui/Button';
import { AppShell } from './AppShell';

const meta: Meta<typeof AppShell> = {
  title: 'Layout/AppShell',
  component: AppShell,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;
type Story = StoryObj<typeof AppShell>;

const sampleContent = (
  <div className="space-y-4">
    <div className="flex items-center gap-2">
      <SeverityTag severity="P1" />
      <span className="text-sm text-zinc-600">Sample incident context</span>
    </div>
    <Button>Primary action</Button>
  </div>
);

export const English: Story = {
  args: { children: sampleContent },
  globals: { locale: 'en' },
};

export const Russian: Story = {
  args: { children: sampleContent },
  globals: { locale: 'ru' },
};
