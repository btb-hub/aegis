import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Select } from './Select';

describe('Select', () => {
  it('renders label and options', () => {
    render(
      <Select
        label="Severity"
        value="critical"
        options={[
          { value: '', label: 'Any' },
          { value: 'critical', label: 'Critical' },
        ]}
        onChange={() => undefined}
      />,
    );
    expect(screen.getByLabelText('Severity')).toHaveValue('critical');
  });

  it('emits changes', () => {
    const onChange = vi.fn();
    render(
      <Select
        label="Status"
        value=""
        options={[{ value: '', label: 'Any' }, { value: 'firing', label: 'Firing' }]}
        onChange={onChange}
      />,
    );
    fireEvent.change(screen.getByLabelText('Status'), { target: { value: 'firing' } });
    expect(onChange).toHaveBeenCalledWith('firing');
  });
});
