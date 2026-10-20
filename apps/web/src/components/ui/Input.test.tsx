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

  it('associates help and validation with the field without changing its accessible name', () => {
    const { rerender } = render(<Input label="Team" value="" onChange={() => undefined} hint="Use the team name" error="Required" />);
    const field = screen.getByRole('textbox', { name: 'Team' });
    expect(field).toHaveAttribute('aria-invalid', 'true');
    expect(field).toHaveAccessibleDescription('Use the team name Required');
    rerender(<Input label="Team" value="Platform" onChange={() => undefined} hint="Use the team name" />);
    expect(field).not.toHaveAttribute('aria-invalid');
    expect(field).toHaveAccessibleDescription('Use the team name');
  });

  it('keeps identically labeled fields independently associated', () => {
    render(<><Input label="Name" value="First" onChange={() => undefined} /><Input label="Name" value="Second" onChange={() => undefined} /></>);
    const fields = screen.getAllByRole('textbox', { name: 'Name' });
    expect(fields[0].id).not.toBe(fields[1].id);
    expect(fields[0]).toHaveValue('First');
    expect(fields[1]).toHaveValue('Second');
  });

  it('disables editing while preserving the field value', () => {
    render(<Input label="Note" value="Please investigate" onChange={() => undefined} disabled />);
    expect(screen.getByRole('textbox', { name: 'Note' })).toBeDisabled();
    expect(screen.getByRole('textbox', { name: 'Note' })).toHaveValue('Please investigate');
  });
});
