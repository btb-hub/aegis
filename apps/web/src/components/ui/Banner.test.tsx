import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Banner } from './Banner';

describe('Banner', () => {
  it('renders warning variant', () => {
    render(<Banner variant="warning">Could not load alerts</Banner>);
    expect(screen.getByRole('alert')).toHaveTextContent('Could not load alerts');
  });
});
