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
  tableClassName?: string;
};

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  emptyMessage,
  compact = false,
  tableClassName = '',
}: DataTableProps<T>) {
  const rowClass = compact ? 'py-2' : 'py-3';

  return (
    <div className="overflow-x-auto rounded-sm border border-zinc-200 bg-white">
      <table className={`min-w-full divide-y divide-zinc-200 text-[13px] ${tableClassName}`.trim()}>
        <thead className="bg-zinc-50 text-left font-mono text-[10px] uppercase tracking-[0.08em] text-zinc-400">
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                scope="col"
                className={`px-4 py-3 font-semibold ${column.headerClassName ?? ''}`.trim()}
              >
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-100">
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
