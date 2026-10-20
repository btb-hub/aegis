import type { Meta, StoryObj } from '@storybook/react';
import { DataTable } from './DataTable';
import { StatusTag } from './StatusTag';

type RecordRow = { id: string; title: string; status: string };
const meta = {
  title: 'UI/DataTable',
  component: DataTable<RecordRow>,
  tags: ['autodocs'],
  args: {
    columns: [
      { key: 'title', header: 'Incident', cellClassName: 'col-span-full', render: (row: RecordRow) => row.title },
      { key: 'status', header: 'Status', cellClassName: 'col-span-full', render: (row: RecordRow) => <StatusTag variant="open" label={row.status} /> },
    ],
    rows: [{ id: '1', title: 'Replication delay in the checkout cluster', status: 'Open' }],
    rowKey: (row: RecordRow) => row.id,
    emptyMessage: 'No incidents match this filter',
  },
} satisfies Meta<typeof DataTable<RecordRow>>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Default: Story = {};
export const PhoneRows: Story = { args: { stackOnMobile: true }, parameters: { viewport: { defaultViewport: 'mobile1' } } };
export const Empty: Story = { args: { rows: [] } };
