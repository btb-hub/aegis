import type { ReactNode } from 'react';

type ButtonVariant = 'primary' | 'secondary' | 'ghost';

type ButtonProps = {
  variant?: ButtonVariant;
  children: ReactNode;
  disabled?: boolean;
  onClick?: () => void;
  type?: 'button' | 'submit';
  href?: string;
  className?: string;
};

const variantClass: Record<ButtonVariant, string> = {
  primary: 'bg-accent text-white hover:bg-blue-700',
  secondary: 'border border-zinc-300 bg-white text-zinc-900 hover:bg-zinc-50',
  ghost: 'text-zinc-700 hover:bg-zinc-100',
};

const baseClass =
  'inline-flex h-9 items-center rounded-md px-3 text-[13px] font-medium disabled:opacity-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent';

export function Button({
  variant = 'primary',
  children,
  disabled = false,
  onClick,
  type = 'button',
  href,
  className = '',
}: ButtonProps) {
  const classes = `${baseClass} ${variantClass[variant]} ${className}`.trim();

  if (href) {
    return (
      <a href={href} className={classes}>
        {children}
      </a>
    );
  }

  return (
    <button type={type} disabled={disabled} onClick={onClick} className={classes}>
      {children}
    </button>
  );
}
