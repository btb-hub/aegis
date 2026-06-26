import type { Meta, StoryObj } from '@storybook/react';
import { SeverityTag } from './SeverityTag';

const meta: Meta<typeof SeverityTag> = {
  title: 'UI/SeverityTag',
  component: SeverityTag,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof SeverityTag>;

export const P1: Story = {
  args: { severity: 'P1' },
};

export const P2: Story = {
  args: { severity: 'P2' },
};

export const P3: Story = {
  args: { severity: 'P3' },
};

export const P4: Story = {
  args: { severity: 'P4' },
};

export const Neutral: Story = {
  args: { severity: 'neutral', label: 'Info' },
};
