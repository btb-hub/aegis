import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { SeverityTag } from './SeverityTag';

describe('SeverityTag', () => {
  it('renders severity code', () => {
    render(<SeverityTag severity="P1" />);
    expect(screen.getByText('P1')).toBeInTheDocument();
  });

  it('renders custom label', () => {
    render(<SeverityTag severity="P2" label="High" />);
    expect(screen.getByText('High')).toBeInTheDocument();
  });
});
