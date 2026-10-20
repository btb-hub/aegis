export type Severity = 'P1' | 'P2' | 'P3' | 'P4' | 'neutral';

const styles: Record<Severity, string> = {
  P1: 'bg-red-50 text-severity-p1',
  P2: 'bg-orange-50 text-severity-p2',
  P3: 'bg-amber-50 text-severity-p3',
  P4: 'bg-blue-50 text-severity-p4',
  neutral: 'bg-zinc-100 text-zinc-700',
};

type SeverityTagProps = {
  severity: Severity;
  label?: string;
};

export function SeverityTag({ severity, label }: SeverityTagProps) {
  return (
    <span className={`inline-flex max-w-full items-center gap-1.5 rounded-[5px] px-2 py-0.5 text-sm font-semibold leading-5 ${styles[severity]}`}>
      {severity !== 'neutral' ? <span aria-hidden="true" className="h-1.5 w-1.5 shrink-0 rounded-full bg-current" /> : null}
      <span className="min-w-0 break-words">{label ?? severity}</span>
    </span>
  );
}
