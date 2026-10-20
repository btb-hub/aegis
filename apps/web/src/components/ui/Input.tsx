import { useId } from 'react';

type InputProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  type?: 'text' | 'password';
  hint?: string;
  autoComplete?: string;
  disabled?: boolean;
};

export function Input({ label, value, onChange, error, type = 'text', hint, autoComplete, disabled = false }: InputProps) {
  const id = useId();
  const describedBy = [hint ? `${id}-hint` : null, error ? `${id}-error` : null].filter(Boolean).join(' ');
  return (
    <div className="text-sm text-zinc-700">
      <label htmlFor={id} className="mb-1 block font-medium">{label}</label>
      <input
        id={id}
        type={type}
        value={value}
        autoComplete={autoComplete}
        disabled={disabled}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy || undefined}
        onChange={(e) => onChange(e.target.value)}
        className="min-h-11 w-full rounded-md border border-zinc-500 bg-white px-3 py-2 text-base text-zinc-900 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/30 disabled:bg-surface-muted disabled:opacity-60 sm:min-h-10 sm:py-1.5"
      />
      {hint ? <span id={`${id}-hint`} className="mt-1 block text-sm text-zinc-600">{hint}</span> : null}
      {error ? <span id={`${id}-error`} className="mt-1 block text-sm text-severity-p1">{error}</span> : null}
    </div>
  );
}
