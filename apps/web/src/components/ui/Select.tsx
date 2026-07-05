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

function SelectChevron() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 16 16"
      className="h-4 w-4 shrink-0 text-zinc-500"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M4 6l4 4 4-4" />
    </svg>
  );
}

const selectClassName =
  'h-9 w-full appearance-none rounded-md border border-zinc-300 bg-white py-0 pl-3 pr-9 text-sm text-zinc-900 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/30 disabled:opacity-50';

export function Select({ label, value, options, onChange, disabled = false, id, hideLabel = false }: SelectProps) {
  const selectId = fieldId(label, id);

  const select = (
    <div className="relative">
      <select
        id={selectId}
        aria-label={hideLabel ? label : undefined}
        value={value}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
        className={selectClassName}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2.5">
        <SelectChevron />
      </span>
    </div>
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
