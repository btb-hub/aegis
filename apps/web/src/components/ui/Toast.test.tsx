import { useState } from 'react';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Toast } from './Toast';
import { Modal } from './Modal';
import '../../i18n';

describe('Toast', () => {
  it('renders message', () => {
    render(<Toast message="Acknowledged" />);
    expect(screen.getByRole('status')).toHaveTextContent('Acknowledged');
  });

  it('renders success variant', () => {
    render(<Toast message="Saved" variant="success" />);
    expect(screen.getByText('Saved')).toHaveClass('border-l-resolved', 'bg-white');
  });

  it('keeps modal error feedback and its recovery actions in the active surface', async () => {
    const retry = vi.fn();
    render(<><Modal title="Edit team" open onClose={() => undefined} primaryLabel="Save" onPrimary={() => undefined}>
      <input aria-label="Name" />
    </Modal><Toast message={<><span>Save failed</span><button onClick={retry}>Retry save</button></>} /></>);
    const dialog = screen.getByRole('dialog', { name: 'Edit team' });
    expect(await within(dialog).findByRole('status')).toHaveTextContent('Save failed');
    expect(within(dialog).getByText('Save failed')).toBeVisible();
    expect(within(dialog).getByRole('button', { name: 'Close' })).toBeEnabled();
    const retryButton = within(dialog).getByRole('button', { name: 'Retry save' });
    retryButton.focus();
    expect(retryButton).toHaveFocus();
    fireEvent.click(retryButton);
    expect(retry).toHaveBeenCalledOnce();
    expect(within(dialog).getByRole('button', { name: 'Save' })).toBeEnabled();
  });

  it('shows a success receipt on the page when saving closes a modal in the same update', () => {
    function Editor() {
      const [open, setOpen] = useState(true);
      const [message, setMessage] = useState('Save failed');
      return <><Modal title="Edit team" open={open} onClose={() => setOpen(false)} primaryLabel="Save" onPrimary={() => {
        setMessage('Team saved');
        setOpen(false);
      }}><input aria-label="Name" /></Modal><Toast message={message} /></>;
    }
    render(<Editor />);
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    const receipt = screen.getByRole('status');
    expect(receipt).toHaveTextContent('Team saved');
    expect(receipt).toBeVisible();
    expect(receipt.parentElement).toBe(document.body);
    expect(receipt.closest('[inert]')).toBeNull();
  });

  it('follows an independently opened and closed dialog without a toast props update', async () => {
    const { container } = render(<Toast message={<a href="/account">Check account</a>} />);
    const dialog = document.createElement('dialog');
    dialog.setAttribute('open', '');
    document.body.append(dialog);
    try {
      await waitFor(() => expect(within(dialog).getByRole('status')).toHaveTextContent('Check account'));
      expect(within(dialog).getByRole('link', { name: 'Check account' })).toHaveAttribute('href', '/account');
      dialog.removeAttribute('open');
      await waitFor(() => expect(screen.getByRole('status').parentElement).toBe(document.body));
      dialog.remove();
      expect(container).toBeEmptyDOMElement();
    } finally {
      dialog.remove();
    }
  });
});
