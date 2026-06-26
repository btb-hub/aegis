import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Toast } from './Toast';

describe('Toast', () => {
  it('renders message', () => {
    render(<Toast message="Acknowledged" />);
    expect(screen.getByRole('status')).toHaveTextContent('Acknowledged');
  });

  it('renders success variant', () => {
    render(<Toast message="Saved" variant="success" />);
    expect(screen.getByRole('status')).toHaveClass('bg-green-50');
  });
});
