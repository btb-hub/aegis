import type { ReactNode } from 'react';

export type SelectOption = {
  value: string;
  label: ReactNode;
};

type SelectProps = {
  label: string;
  value: string;
  options: SelectOption[];
  onChange: (value: string) => void;
  disabled?: boolean;
  id?: string;
  hideLabel?: boolean;
};

function fieldId(label: string, id?: string) {
  return id ?? label.toLowerCase().replace(/\s+/g, '-');
}

export function Select({ label, value, options, onChange, disabled = false, id, hideLabel = false }: SelectProps) {
  const selectId = fieldId(label, id);

  const select = (
    <select
      id={selectId}
      aria-label={hideLabel ? label : undefined}
      value={value}
      disabled={disabled}
      onChange={(event) => onChange(event.target.value)}
      className="h-9 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-900 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/30 disabled:opacity-50"
    >
      {options.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  );

  if (hideLabel) {
    return select;
  }

  return (
    <label className="block text-sm text-zinc-700" htmlFor={selectId}>
      <span className="mb-1 block font-medium">{label}</span>
      {select}
    </label>
  );
}
