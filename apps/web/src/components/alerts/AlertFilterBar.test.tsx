import { fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { defaultAlertFilters, type AlertFilters } from '../../lib/alertTypes';
import { AlertFilterBar } from './AlertFilterBar';

function StatefulFilterBar() {
  const [filters, setFilters] = useState<AlertFilters>(defaultAlertFilters());
  return (
    <AlertFilterBar
      filters={filters}
      onChange={setFilters}
      onApply={vi.fn()}
    />
  );
}

describe('AlertFilterBar', () => {
  it('updates filters including label grouping controls', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <StatefulFilterBar />
      </I18nextProvider>,
    );

    fireEvent.change(screen.getByLabelText('Search'), { target: { value: 'cpu' } });
    fireEvent.change(screen.getByLabelText('Severity'), { target: { value: 'critical' } });
    fireEvent.change(screen.getByLabelText('Status'), { target: { value: 'firing' } });
    fireEvent.change(screen.getByLabelText('Key'), { target: { value: 'team' } });
    fireEvent.change(screen.getByLabelText('Value'), { target: { value: 'platform' } });
    fireEvent.change(screen.getByLabelText('Group by'), { target: { value: 'label' } });
    fireEvent.change(screen.getByLabelText('Label key'), { target: { value: 'service' } });
    fireEvent.click(screen.getByRole('button', { name: 'Apply filters' }));

    expect(screen.getByLabelText('Search')).toHaveValue('cpu');
    expect(screen.getByLabelText('Label key')).toHaveValue('service');
  });
});
