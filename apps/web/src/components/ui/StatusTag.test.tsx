import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { StatusTag, alertStatusVariant, incidentStatusVariant } from './StatusTag';

describe('StatusTag', () => {
  it('renders status label', () => {
    render(<StatusTag variant="open" label="Open" />);
    expect(screen.getByText('Open')).toBeInTheDocument();
  });
});

describe('status variant helpers', () => {
  it('maps incident statuses directly', () => {
    expect(incidentStatusVariant('acknowledged')).toBe('acknowledged');
  });

  it('maps alert firing status', () => {
    expect(alertStatusVariant('firing')).toBe('firing');
  });
});
