import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Button } from './Button';

describe('Button', () => {
  it('renders primary label', () => {
    render(<Button>Acknowledge</Button>);
    expect(screen.getByRole('button', { name: 'Acknowledge' })).toBeInTheDocument();
  });

  it('calls onClick', () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Go</Button>);
    fireEvent.click(screen.getByRole('button', { name: 'Go' }));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it('disables interaction', () => {
    render(<Button disabled>Go</Button>);
    expect(screen.getByRole('button', { name: 'Go' })).toBeDisabled();
  });

  it('renders as a link when href is provided', () => {
    render(
      <Button href="/auth/google/login" variant="secondary">
        Sign in
      </Button>,
    );
    expect(screen.getByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/auth/google/login');
  });

  it('opens external hrefs in a new tab', () => {
    render(
      <Button href="https://example.com" target="_blank" rel="noopener noreferrer">
        Open
      </Button>,
    );
    const link = screen.getByRole('link', { name: 'Open' });
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noopener noreferrer');
  });
});
