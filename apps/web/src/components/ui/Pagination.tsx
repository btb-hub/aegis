import { Button } from './Button';

type PaginationProps = {
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  totalLabel: string;
  prevLabel: string;
  nextLabel: string;
  pageLabel: string;
};

export function Pagination({
  page,
  pageSize,
  total,
  onPageChange,
  totalLabel,
  prevLabel,
  nextLabel,
  pageLabel,
}: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 text-sm text-zinc-600">
      <span>{totalLabel}</span>
      <div className="flex items-center gap-2">
        <Button variant="ghost" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
          {prevLabel}
        </Button>
        <span>{pageLabel}</span>
        <Button variant="ghost" disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}>
          {nextLabel}
        </Button>
      </div>
    </div>
  );
}
