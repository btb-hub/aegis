import type { ReactNode } from 'react';

type ButtonVariant = 'primary' | 'secondary' | 'ghost';

type ButtonProps = {
  variant?: ButtonVariant;
  children: ReactNode;
  disabled?: boolean;
  onClick?: () => void;
  type?: 'button' | 'submit';
};

const variantClass: Record<ButtonVariant, string> = {
  primary: 'bg-accent text-white hover:bg-blue-700',
  secondary: 'border border-zinc-300 bg-white text-zinc-900 hover:bg-zinc-50',
  ghost: 'text-zinc-700 hover:bg-zinc-100',
};

export function Button({
  variant = 'primary',
  children,
  disabled = false,
  onClick,
  type = 'button',
}: ButtonProps) {
  return (
    <button
      type={type}
      disabled={disabled}
      onClick={onClick}
      className={`inline-flex h-9 items-center rounded-md px-3 text-[13px] font-medium disabled:opacity-50 ${variantClass[variant]}`}
    >
      {children}
    </button>
  );
}
