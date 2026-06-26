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
      className={`fixed right-4 top-4 z-50 rounded-md border px-4 py-3 text-sm shadow-sm ${variantClass[variant]}`}
    >
      {message}
    </div>
  );
}
