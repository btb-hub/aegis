import type { ButtonHTMLAttributes, ReactNode } from 'react';

type ButtonVariant = 'primary' | 'secondary' | 'ghost';

type ButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> & {
  variant?: ButtonVariant;
  children: ReactNode;
  disabled?: boolean;
  onClick?: () => void;
  type?: 'button' | 'submit';
  href?: string;
  target?: string;
  rel?: string;
  className?: string;
};

const variantClass: Record<ButtonVariant, string> = {
  primary: 'bg-accent text-white hover:bg-blue-700',
  secondary: 'border border-zinc-300 bg-white text-zinc-900 hover:bg-zinc-50',
  ghost: 'text-accent hover:bg-blue-50',
};

const baseClass =
  'inline-flex min-h-11 items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium leading-5 transition-colors sm:min-h-9 disabled:cursor-not-allowed disabled:opacity-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent';

export function Button({
  variant = 'primary',
  children,
  disabled = false,
  onClick,
  type = 'button',
  href,
  target,
  rel,
  className = '',
  ...props
}: ButtonProps) {
  const classes = `${baseClass} ${variantClass[variant]} ${className}`.trim();

  if (href) {
    return (
      <a href={disabled ? undefined : href} aria-disabled={disabled || undefined} tabIndex={disabled ? -1 : undefined} className={classes} target={target} rel={rel}>
        {children}
      </a>
    );
  }

  return (
    <button {...props} type={type} disabled={disabled} onClick={onClick} className={classes}>
      {children}
    </button>
  );
}
