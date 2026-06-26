export type Severity = 'P1' | 'P2' | 'P3' | 'P4' | 'neutral';

const styles: Record<Severity, string> = {
  P1: 'bg-red-100 text-severity-p1',
  P2: 'bg-orange-100 text-severity-p2',
  P3: 'bg-amber-100 text-severity-p3',
  P4: 'bg-purple-100 text-severity-p4',
  neutral: 'bg-zinc-100 text-zinc-700',
};

type SeverityTagProps = {
  severity: Severity;
  label?: string;
};

export function SeverityTag({ severity, label }: SeverityTagProps) {
  return (
    <span className={`inline-flex rounded px-2 py-0.5 font-mono text-xs font-medium ${styles[severity]}`}>
      {label ?? severity}
    </span>
  );
}
