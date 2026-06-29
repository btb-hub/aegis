import type { ReactNode } from 'react';

type ToastProps = {
  message: ReactNode;
  variant?: 'default' | 'success';
};

const variantClass = {
  default: 'border-zinc-200 bg-white text-zinc-900',
  success: 'border-green-200 bg-green-50 text-green-900',
};

export function Toast({ message, variant = 'default' }: ToastProps) {
  return (
    <div
      role="status"
      className="pointer-events-none fixed inset-x-4 top-16 z-50 flex justify-end sm:inset-x-auto sm:right-6"
    >
      <div
        className={`pointer-events-auto max-w-sm rounded-md border px-4 py-3 text-sm shadow-md ${variantClass[variant]}`}
      >
        {message}
      </div>
    </div>
  );
}
