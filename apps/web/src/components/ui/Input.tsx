type InputProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
};

export function Input({ label, value, onChange, error }: InputProps) {
  const id = label.toLowerCase().replace(/\s+/g, '-');
  return (
    <label className="block text-sm text-zinc-700" htmlFor={id}>
      <span className="mb-1 block font-medium">{label}</span>
      <input
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="h-9 w-full rounded-md border border-zinc-300 px-3 text-sm focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/30"
      />
      {error ? <span className="mt-1 block text-xs text-severity-p1">{error}</span> : null}
    </label>
  );
}
