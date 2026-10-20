import { fireEvent, render, screen, within } from '@testing-library/react';
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

  it('focuses the form, contains keyboard focus, and provides a visible close control', () => {
    const onClose = vi.fn();
    render(<Modal title="Edit team" open onClose={onClose} primaryLabel="Save" onPrimary={() => undefined}>
      <label>Name<input defaultValue="Platform" /></label><button disabled>Unavailable</button>
    </Modal>);
    const dialog = screen.getByRole('dialog', { name: 'Edit team' });
    expect(screen.getByRole('textbox', { name: 'Name' })).toHaveFocus();
    const close = within(dialog).getByRole('button', { name: 'Close' });
    close.focus();
    fireEvent.keyDown(close, { key: 'Tab', shiftKey: true });
    expect(screen.getByRole('button', { name: 'Save' })).toHaveFocus();
    fireEvent.keyDown(document.activeElement!, { key: 'Tab' });
    expect(close).toHaveFocus();
    fireEvent.click(close);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('restores invoker focus, background state, and scrolling when closed', () => {
    document.body.style.overflow = 'auto';
    const { rerender } = render(<button>Open editor</button>);
    const trigger = screen.getByRole('button', { name: 'Open editor' });
    trigger.focus();
    const renderContent = (open: boolean) => <><button>Open editor</button><Modal title="Edit" open={open} onClose={() => undefined}><p>Details</p></Modal></>;
    rerender(renderContent(true));
    expect(document.body.style.overflow).toBe('hidden');
    expect(trigger.closest('[inert]')).not.toBeNull();
    trigger.focus();
    expect(screen.getByRole('button', { name: 'Close' })).toHaveFocus();
    rerender(renderContent(false));
    expect(document.body.style.overflow).toBe('auto');
    expect(trigger.closest('[inert]')).toBeNull();
    expect(trigger).toHaveFocus();
    document.body.style.overflow = '';
  });

  it('keeps form focus stable when the close callback changes during editing', () => {
    const oldClose = vi.fn();
    const newClose = vi.fn();
    const { rerender } = render(<Modal title="Edit" open onClose={oldClose}><input aria-label="Name" /><input aria-label="Description" /></Modal>);
    const description = screen.getByRole('textbox', { name: 'Description' });
    description.focus();
    rerender(<Modal title="Edit" open onClose={newClose}><input aria-label="Name" /><input aria-label="Description" /></Modal>);
    expect(description).toHaveFocus();
    fireEvent.keyDown(description, { key: 'Escape' });
    expect(newClose).toHaveBeenCalledOnce();
    expect(oldClose).not.toHaveBeenCalled();
  });
});
