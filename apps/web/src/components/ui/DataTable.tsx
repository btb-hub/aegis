import type { ReactNode } from 'react';

export type DataTableColumn<T> = {
  key: string;
  header: string;
  render: (row: T) => ReactNode;
  headerClassName?: string;
  cellClassName?: string;
};

type DataTableProps<T> = {
  columns: DataTableColumn<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  emptyMessage: string;
  compact?: boolean;
};

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  emptyMessage,
  compact = false,
}: DataTableProps<T>) {
  const rowClass = compact ? 'py-2' : 'py-3';

  return (
    <div className="overflow-hidden rounded-lg border border-zinc-200 bg-white">
      <table className="min-w-full divide-y divide-zinc-200 text-sm">
        <thead className="bg-zinc-50 text-left text-zinc-600">
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                className={`px-4 py-3 font-medium ${column.headerClassName ?? ''}`.trim()}
              >
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200">
          {rows.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-4 py-8 text-center text-zinc-600">
                {emptyMessage}
              </td>
            </tr>
          ) : (
            rows.map((row) => (
              <tr key={rowKey(row)} className="hover:bg-zinc-50">
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={`px-4 ${rowClass} ${column.cellClassName ?? ''}`.trim()}
                  >
                    {column.render(row)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
