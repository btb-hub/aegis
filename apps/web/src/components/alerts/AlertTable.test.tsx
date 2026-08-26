import { fireEvent, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import type { AlertGroup, AlertItem } from '../../lib/alertTypes';
import { AlertGroupTable, AlertTable } from './AlertTable';

const alert: AlertItem = {
  id: 'alert-1',
  fingerprint: 'fp-1',
  status: 'firing',
  severity: 'critical',
  title: 'CPU high',
  labels: {},
  received_at: '2026-06-26T10:00:00Z',
  incident_id: null,
};

function renderWithI18n(ui: ReactNode) {
  return render(<I18nextProvider i18n={i18n}>{ui}</I18nextProvider>);
}

describe('AlertTable', () => {
  it('renders empty state', () => {
    renderWithI18n(
      <AlertTable items={[]} total={0} page={1} pageSize={25} onPageChange={vi.fn()} />,
    );
    expect(screen.getByText('No alerts match these filters')).toBeInTheDocument();
  });

  it('renders rows and paginates', () => {
    const onPageChange = vi.fn();
    renderWithI18n(
      <AlertTable items={[alert]} total={75} page={2} pageSize={25} onPageChange={onPageChange} />,
    );

    expect(screen.getByText('CPU high')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Previous' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    expect(onPageChange).toHaveBeenCalledWith(1);
    expect(onPageChange).toHaveBeenCalledWith(3);
  });
});

describe('AlertGroupTable', () => {
  it('renders grouped buckets and empty sample placeholder', () => {
    const groups: AlertGroup[] = [
      { key: 'critical', count: 2, sample: alert },
      { key: '', count: 1 },
    ];

    renderWithI18n(<AlertGroupTable groups={groups} groupBy="severity" total={3} />);

    expect(screen.getByText('critical')).toBeInTheDocument();
    expect(screen.getByText(/CPU high/)).toBeInTheDocument();
    expect(screen.getByText('(empty)')).toBeInTheDocument();
  });

  it('renders empty grouped state', () => {
    renderWithI18n(<AlertGroupTable groups={[]} groupBy="severity" total={0} />);
    expect(screen.getByText('No alerts match these filters')).toBeInTheDocument();
  });
});
