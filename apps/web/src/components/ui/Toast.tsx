import type { ReactNode } from 'react';

type ToastProps = {
  message: ReactNode;
  variant?: 'default' | 'success';
};

const variantClass = {
  default: 'border-zinc-200 border-l-zinc-400 bg-white text-zinc-900',
  success: 'border-zinc-200 border-l-resolved bg-white text-zinc-900',
};

export function Toast({ message, variant = 'default' }: ToastProps) {
  return (
    <div
      role="status"
      className="pointer-events-none fixed inset-x-4 top-16 z-50 flex justify-end sm:inset-x-auto sm:right-6"
    >
      <div
        className={`pointer-events-auto max-w-sm rounded-md border border-l-[3px] px-4 py-3 text-[13px] shadow-md ${variantClass[variant]}`}
      >
        {message}
      </div>
    </div>
  );
}
