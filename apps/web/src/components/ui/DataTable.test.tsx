import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DataTable } from './DataTable';

describe('DataTable', () => {
  it('renders rows and empty state', () => {
    const { rerender } = render(
      <DataTable
        columns={[{ key: 'name', header: 'Name', render: (row: { name: string }) => row.name }]}
        rows={[{ name: 'Platform' }]}
        rowKey={(row) => row.name}
        emptyMessage="No rows"
      />,
    );
    expect(screen.getByText('Platform')).toBeInTheDocument();

    rerender(
      <DataTable
        columns={[{ key: 'name', header: 'Name', render: (row: { name: string }) => row.name }]}
        rows={[]}
        rowKey={(row) => row.name}
        emptyMessage="No rows"
      />,
    );
    expect(screen.getByText('No rows')).toBeInTheDocument();
  });
});
