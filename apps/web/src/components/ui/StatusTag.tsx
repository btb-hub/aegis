import type { IncidentStatus } from '../../lib/incidentTypes';

export type StatusVariant = 'open' | 'acknowledged' | 'resolved' | 'firing' | 'neutral';

const styles: Record<StatusVariant, string> = {
  open: 'bg-red-50 text-severity-p1',
  acknowledged: 'bg-amber-50 text-severity-p3',
  resolved: 'bg-green-50 text-resolved',
  firing: 'bg-red-50 text-severity-p1',
  neutral: 'bg-zinc-100 text-zinc-700',
};

type StatusTagProps = {
  variant: StatusVariant;
  label: string;
};

export function StatusTag({ variant, label }: StatusTagProps) {
  return (
    <span className={`inline-flex w-fit max-w-full items-center gap-1.5 rounded-full px-2 py-0.5 text-sm font-medium leading-5 ${styles[variant]}`}>
      <span aria-hidden="true" className="h-1.5 w-1.5 shrink-0 rounded-full bg-current" />
      <span className="min-w-0 break-words">{label}</span>
    </span>
  );
}

export function incidentStatusVariant(status: IncidentStatus): StatusVariant {
  return status;
}

export function alertStatusVariant(status: string): StatusVariant {
  switch (status.toLowerCase()) {
    case 'firing':
      return 'firing';
    case 'resolved':
      return 'resolved';
    default:
      return 'neutral';
  }
}
