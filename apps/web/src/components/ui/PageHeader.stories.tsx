import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { Button } from './Button';
import { PageHeader } from './PageHeader';

const meta: Meta<typeof PageHeader> = {
  title: 'UI/PageHeader',
  component: PageHeader,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <MemoryRouter>
        <Story />
      </MemoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof PageHeader>;

export const Default: Story = {
  args: {
    title: 'Alerts',
    subtitle: 'Search, filter, group, and export alert history',
    breadcrumb: {
      ariaLabel: 'Breadcrumb',
      items: [{ label: 'Platform', href: '/dashboard' }, { label: 'Alerts' }],
    },
  },
};

export const WithActions: Story = {
  args: {
    ...Default.args,
    actions: <Button variant="secondary">Export CSV</Button>,
  },
};
