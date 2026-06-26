import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Input } from './Input';

describe('Input', () => {
  it('renders label and value', () => {
    render(<Input label="Team name" value="platform" onChange={() => undefined} />);
    expect(screen.getByLabelText('Team name')).toHaveValue('platform');
  });

  it('emits changes', () => {
    const onChange = vi.fn();
    render(<Input label="Team" value="" onChange={onChange} />);
    fireEvent.change(screen.getByLabelText('Team'), { target: { value: 'a' } });
    expect(onChange).toHaveBeenCalledWith('a');
  });

  it('shows error', () => {
    render(<Input label="Team" value="" onChange={() => undefined} error="Required" />);
    expect(screen.getByText('Required')).toBeInTheDocument();
  });
});
