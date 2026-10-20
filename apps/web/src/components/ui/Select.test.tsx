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

  it('generates unique label associations and respects explicit identifiers', () => {
    const options = [{ value: 'all', label: 'All' }];
    render(<><Select label="Status" value="all" options={options} onChange={() => undefined} /><Select label="Status" value="all" options={options} onChange={() => undefined} /><Select label="Scope" id="scope-filter" hideLabel value="all" options={options} onChange={() => undefined} /></>);
    const fields = screen.getAllByRole('combobox', { name: 'Status' });
    expect(fields[0].id).not.toBe(fields[1].id);
    expect(screen.getByRole('combobox', { name: 'Scope' })).toHaveAttribute('id', 'scope-filter');
  });
});
