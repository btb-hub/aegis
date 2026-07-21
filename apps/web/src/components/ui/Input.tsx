type InputProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  type?: 'text' | 'password';
  hint?: string;
  autoComplete?: string;
};

export function Input({ label, value, onChange, error, type = 'text', hint, autoComplete }: InputProps) {
  const id = label.toLowerCase().replace(/\s+/g, '-');
  return (
    <label className="block text-sm text-zinc-700" htmlFor={id}>
      <span className="mb-1 block font-medium">{label}</span>
      <input
        id={id}
        type={type}
        value={value}
        autoComplete={autoComplete}
        onChange={(e) => onChange(e.target.value)}
        className="h-9 w-full rounded-md border border-zinc-300 px-3 text-sm focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/30"
      />
      {hint ? <span className="mt-1 block text-xs text-zinc-500">{hint}</span> : null}
      {error ? <span className="mt-1 block text-xs text-severity-p1">{error}</span> : null}
    </label>
  );
}
