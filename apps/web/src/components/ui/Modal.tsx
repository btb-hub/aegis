import { useEffect, useId, useRef, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { useTranslation } from 'react-i18next';
import { Button } from './Button';

type ModalProps = {
  title: string;
  children: ReactNode;
  open: boolean;
  onClose: () => void;
  primaryLabel?: string;
  secondaryLabel?: string;
  onPrimary?: () => void;
  primaryDisabled?: boolean;
  primaryLoading?: boolean;
};

export function Modal({
  title,
  children,
  open,
  onClose,
  primaryLabel,
  secondaryLabel,
  onPrimary,
  primaryDisabled = false,
  primaryLoading = false,
}: ModalProps) {
  const { t } = useTranslation();
  const titleId = useId();
  const dialogRef = useRef<HTMLDialogElement>(null);
  const onCloseRef = useRef(onClose);

  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!open || !dialog) {
      return;
    }
    const invoker = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    // Native modal dialogs make the rest of the document inert. Keep a fallback
    // for hosts that don't implement showModal (including the unit-test DOM).
    const background: Array<{ element: Element; inert: boolean }> = [];
    if (typeof dialog.showModal === 'function') {
      dialog.showModal();
    } else {
      dialog.setAttribute('open', '');
      for (const element of document.body.children) {
        if (element !== dialog) {
          background.push({ element, inert: element.hasAttribute('inert') });
          element.setAttribute('inert', '');
        }
      }
    }

    const focusableElements = () => Array.from(dialog.querySelectorAll<HTMLElement>(
      'a[href], button, input, select, textarea, [tabindex], [contenteditable="true"]',
    )).filter((element) => element.tabIndex >= 0
      && !element.matches(':disabled, input[type="hidden"]')
      && !element.closest('[hidden], [inert]')
      && getComputedStyle(element).display !== 'none'
      && getComputedStyle(element).visibility !== 'hidden');

    const initialTarget = focusableElements().find((element) => element.matches('input, select, textarea'));
    (initialTarget ?? dialog.querySelector<HTMLElement>('h2') ?? dialog).focus();

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        onCloseRef.current();
      } else if (event.key === 'Tab') {
        const elements = focusableElements();
        const first = elements[0];
        const last = elements[elements.length - 1];
        const active = document.activeElement;
        if (!first) {
          event.preventDefault();
          dialog.focus();
        } else if (event.shiftKey && (active === first || !elements.includes(active as HTMLElement))) {
          event.preventDefault();
          last.focus();
        } else if (!event.shiftKey && (active === last || !elements.includes(active as HTMLElement))) {
          event.preventDefault();
          first.focus();
        }
      }
    };
    const containFocus = (event: FocusEvent) => {
      if (event.target instanceof Node && !dialog.contains(event.target)) {
        (focusableElements()[0] ?? dialog).focus();
      }
    };
    document.addEventListener('keydown', onKeyDown);
    document.addEventListener('focusin', containFocus);
    return () => {
      document.removeEventListener('keydown', onKeyDown);
      document.removeEventListener('focusin', containFocus);
      if (typeof dialog.close === 'function') {
        dialog.close();
      } else {
        dialog.removeAttribute('open');
      }
      for (const { element, inert } of background) {
        if (!inert) {
          element.removeAttribute('inert');
        }
      }
      document.body.style.overflow = previousOverflow;
      if (invoker?.isConnected && !invoker.closest('[inert]')) {
        invoker.focus();
      }
    };
  }, [open]);

  if (!open) {
    return null;
  }

  return createPortal(
    <dialog
      ref={dialogRef}
      tabIndex={-1}
      aria-modal="true"
      aria-labelledby={titleId}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
      onClick={(event) => {
        if (event.target !== event.currentTarget) {
          return;
        }
        const bounds = event.currentTarget.getBoundingClientRect();
        if (event.clientX < bounds.left || event.clientX > bounds.right
          || event.clientY < bounds.top || event.clientY > bounds.bottom) {
          onClose();
        }
      }}
      className="fixed inset-0 m-auto max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] max-w-md overflow-y-auto rounded-xl border border-zinc-200 bg-white p-0 text-zinc-900 shadow-2xl backdrop:bg-zinc-950/45"
    >
      <div className="flex items-start justify-between gap-4 border-b border-zinc-200 px-5 py-4">
        <h2 id={titleId} tabIndex={-1} className="pt-2 text-lg font-semibold text-zinc-900 focus:outline-none">
          {title}
        </h2>
        <button
          type="button"
          onClick={onClose}
          aria-label={t('modal.close')}
          className="flex h-11 w-11 shrink-0 items-center justify-center rounded-md text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent sm:h-10 sm:w-10"
        >
          <svg aria-hidden="true" viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
            <path d="m6 6 12 12M18 6 6 18" />
          </svg>
        </button>
      </div>
      <div className="space-y-4 px-5 py-4">{children}</div>
      {primaryLabel || secondaryLabel ? (
        <div className="flex flex-wrap justify-end gap-2 border-t border-zinc-200 px-5 py-3.5">
          {secondaryLabel ? (
            <Button variant="secondary" onClick={onClose}>
              {secondaryLabel}
            </Button>
          ) : null}
          {primaryLabel && onPrimary ? (
            <Button disabled={primaryDisabled || primaryLoading} onClick={onPrimary}>
              {primaryLoading ? t('teams.saving') : primaryLabel}
            </Button>
          ) : null}
        </div>
      ) : null}
    </dialog>,
    document.body,
  );
}
