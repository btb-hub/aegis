import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Checkbox } from './Checkbox';

describe('Checkbox', () => {
  it('renders checked state', () => {
    render(<Checkbox label="Share with team" checked onChange={() => undefined} />);
    expect(screen.getByLabelText('Share with team')).toBeChecked();
  });

  it('emits changes', () => {
    const onChange = vi.fn();
    render(<Checkbox label="Share with team" checked={false} onChange={onChange} />);
    fireEvent.click(screen.getByLabelText('Share with team'));
    expect(onChange).toHaveBeenCalledWith(true);
  });
});
