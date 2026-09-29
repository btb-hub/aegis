import { fireEvent, render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { ShiftsCalendar } from './ShiftsCalendar';

describe('ShiftsCalendar', () => {
  it('renders rotation and override entries', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <ShiftsCalendar
          month={new Date('2026-06-01T00:00:00Z')}
          slots={[
            {
              id: 's1',
              userId: 'u1',
              displayName: 'Alice',
              startAt: '2026-06-02T00:00:00Z',
              endAt: '2026-06-09T00:00:00Z',
              source: 'rotation',
            },
          ]}
          overrides={[
            {
              id: 'o1',
              userId: 'u2',
              displayName: 'Bob',
              startAt: '2026-06-10T00:00:00Z',
              endAt: '2026-06-11T00:00:00Z',
            },
          ]}
        />
      </I18nextProvider>,
    );
    expect(screen.getAllByText('Alice').length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: 'Next period' }));
    expect(screen.getAllByText('Bob').length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: 'Month' }));
    expect(screen.getByRole('button', { name: 'Month' })).toHaveAttribute('aria-pressed', 'true');
  });

  it('places teams and spanning assignments in the week timeline', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <ShiftsCalendar
          month={new Date('2026-06-24T00:00:00Z')}
          lanes={[
            { id: 'payments', name: 'Payments', slots: [{ id: 's1', userId: 'u1', displayName: 'Jamie Lee', startAt: '2026-06-22T00:00:00Z', endAt: '2026-06-29T00:00:00Z', source: 'rotation' }], overrides: [{ id: 'o1', userId: 'u2', displayName: 'Kit T', startAt: '2026-06-24T00:00:00Z', endAt: '2026-06-25T00:00:00Z' }] },
            { id: 'search', name: 'Search', slots: [], overrides: [] },
          ]}
        />
      </I18nextProvider>,
    );
    expect(screen.getByText('Payments')).toBeInTheDocument();
    expect(screen.getByText('Search')).toBeInTheDocument();
    expect(screen.getByTitle('Jamie Lee · Rotation')).toHaveStyle({ left: '0%', width: '100%' });
    expect(screen.getByTitle('Kit T · Override')).toBeInTheDocument();
  });

  it('navigates a controlled calendar and opens the override action', () => {
    const onDateChange = vi.fn();
    const onAddOverride = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <ShiftsCalendar month={new Date('2026-06-24T00:00:00Z')} onDateChange={onDateChange} onAddOverride={onAddOverride} />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Next period' }));
    expect(onDateChange).toHaveBeenCalledWith(new Date('2026-07-01T00:00:00Z'));

    fireEvent.click(screen.getByRole('button', { name: 'Add override' }));
    expect(onAddOverride).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByRole('button', { name: 'Today' }));
    expect(onDateChange.mock.lastCall?.[0].toDateString()).toBe(new Date().toDateString());

    fireEvent.click(screen.getByRole('button', { name: 'Month' }));
    fireEvent.click(screen.getByRole('button', { name: 'Previous period' }));
    expect(onDateChange).toHaveBeenLastCalledWith(new Date('2026-05-24T00:00:00Z'));
  });
});
