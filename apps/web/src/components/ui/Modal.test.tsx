import { fireEvent, render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { Modal } from './Modal';

describe('Modal', () => {
  it('renders dialog content and handles actions', () => {
    const onClose = vi.fn();
    const onPrimary = vi.fn();

    render(
      <I18nextProvider i18n={i18n}>
        <Modal
          title="Create team"
          open
          onClose={onClose}
          primaryLabel="Save"
          secondaryLabel="Cancel"
          onPrimary={onPrimary}
        >
          <p>Form body</p>
        </Modal>
      </I18nextProvider>,
    );

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('Form body')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(onPrimary).toHaveBeenCalledTimes(1);
  });

  it('closes on escape and shows loading label on primary action', () => {
    const onClose = vi.fn();

    render(
      <I18nextProvider i18n={i18n}>
        <Modal
          title="Create team"
          open
          onClose={onClose}
          primaryLabel="Save"
          secondaryLabel="Cancel"
          onPrimary={() => undefined}
          primaryLoading
        >
          <p>Form body</p>
        </Modal>
      </I18nextProvider>,
    );

    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('button', { name: 'Saving' })).toBeDisabled();
  });

  it('renders nothing when closed', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <Modal title="Hidden" open={false} onClose={() => undefined}>
          <p>Hidden body</p>
        </Modal>
      </I18nextProvider>,
    );

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
