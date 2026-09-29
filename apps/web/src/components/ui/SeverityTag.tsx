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
    <span className={`inline-flex items-center gap-1.5 rounded-[5px] px-2 py-1 font-mono text-[11px] font-semibold leading-4 ${styles[severity]}`}>
      {severity !== 'neutral' ? <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-current" /> : null}
      {label ?? severity}
    </span>
  );
}
