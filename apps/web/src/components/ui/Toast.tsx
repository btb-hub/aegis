import { useLayoutEffect, useRef, useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

type ToastProps = {
  message: ReactNode;
  variant?: 'default' | 'success';
};

const variantClass = {
  default: 'border-zinc-200 border-l-zinc-400 bg-white text-zinc-900',
  success: 'border-zinc-200 border-l-resolved bg-white text-zinc-900',
};

function activeSurface(): HTMLElement {
  const dialogs = document.querySelectorAll<HTMLDialogElement>('dialog[open]');
  return dialogs[dialogs.length - 1] ?? document.body;
}

export function Toast({ message, variant = 'default' }: ToastProps) {
  const [surface, setSurface] = useState<HTMLElement>(document.body);
  const receiptRef = useRef<HTMLDivElement>(null);

  // Resolve after React commits so a success that closes a modal in the same
  // update can't leave its receipt inside the detached dialog.
  useLayoutEffect(() => {
    setSurface(activeSurface());
  }, [message, variant]);

  useLayoutEffect(() => {
    const observer = new MutationObserver(() => setSurface(activeSurface()));
    observer.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ['open'],
    });
    return () => observer.disconnect();
  }, []);

  useLayoutEffect(() => {
    if (surface !== document.body) {
      receiptRef.current?.scrollIntoView?.({ block: 'nearest' });
    }
  }, [message, surface]);

  const inDialog = surface !== document.body;
  return createPortal(
    <div
      ref={receiptRef}
      role="status"
      className={inDialog
        ? 'border-t border-zinc-200 bg-white px-5 py-4'
        : 'pointer-events-none fixed inset-x-4 top-16 z-50 flex justify-end sm:inset-x-auto sm:right-6'}
    >
      <div
        className={`pointer-events-auto max-w-sm rounded-md border border-l-[3px] px-4 py-3 text-[13px] shadow-md ${variantClass[variant]}`}
      >
        {message}
      </div>
    </div>,
    surface,
  );
}
