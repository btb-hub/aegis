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
  stackOnMobile?: boolean;
};

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  emptyMessage,
  compact = false,
  tableClassName = '',
  stackOnMobile = false,
}: DataTableProps<T>) {
  const rowClass = compact ? 'py-2' : 'py-3';

  return (
    <div className="overflow-x-auto rounded-lg border border-zinc-200 bg-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent" tabIndex={0}>
      <table role="table" className={`min-w-full divide-y divide-zinc-200 text-sm tabular-nums ${stackOnMobile ? 'block sm:table' : ''} ${tableClassName}`.trim()}>
        <thead className={`bg-surface-muted text-left text-sm text-zinc-600 ${stackOnMobile ? 'max-sm:sr-only' : ''}`}>
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
        <tbody className={`divide-y divide-zinc-100 ${stackOnMobile ? 'block sm:table-row-group' : ''}`}>
          {rows.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-4 py-8 text-center text-zinc-600">
                {emptyMessage}
              </td>
            </tr>
          ) : (
            rows.map((row) => (
              <tr key={rowKey(row)} className={`hover:bg-zinc-50 ${stackOnMobile ? 'grid grid-cols-[repeat(auto-fit,minmax(min(100%,max(6rem,calc(50%_-_0.375rem))),1fr))] gap-x-3 px-[16px] py-3 sm:table-row sm:p-0' : ''}`}>
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={`${stackOnMobile ? 'block min-w-0 py-1 sm:table-cell sm:px-4 sm:py-3' : `px-4 ${rowClass}`} ${column.cellClassName ?? ''}`.trim()}
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
