import { useEffect, useId, type ReactNode } from 'react';
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

  useEffect(() => {
    if (!open) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [open, onClose]);

  if (!open) {
    return null;
  }

  return createPortal(
    <div className="fixed inset-0 z-50">
      <button
        type="button"
        aria-label={t('modal.close')}
        className="absolute inset-0 bg-zinc-900/40"
        onClick={onClose}
      />
      <div className="pointer-events-none flex min-h-full items-center justify-center p-4">
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          className="pointer-events-auto relative z-10 w-full max-w-md rounded-lg border border-zinc-200 bg-white p-6 shadow-lg"
        >
          <h2 id={titleId} className="text-lg font-semibold text-zinc-900">
            {title}
          </h2>
          <div className="mt-4 space-y-4">{children}</div>
          {primaryLabel || secondaryLabel ? (
            <div className="mt-6 flex justify-end gap-2">
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
        </div>
      </div>
    </div>,
    document.body,
  );
}
