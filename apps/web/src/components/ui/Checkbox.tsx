type CheckboxProps = {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  id?: string;
};

function fieldId(label: string, id?: string) {
  return id ?? label.toLowerCase().replace(/\s+/g, '-');
}

export function Checkbox({ label, checked, onChange, disabled = false, id }: CheckboxProps) {
  const checkboxId = fieldId(label, id);

  return (
    <label
      htmlFor={checkboxId}
      className="flex min-h-9 cursor-pointer items-center gap-2 text-sm text-zinc-700"
    >
      <input
        id={checkboxId}
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
        className="h-4 w-4 rounded border-zinc-300 text-accent focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:opacity-50"
      />
      {label}
    </label>
  );
}
