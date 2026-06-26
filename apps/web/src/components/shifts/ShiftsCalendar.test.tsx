import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it } from 'vitest';
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
    expect(screen.getAllByText('Bob').length).toBeGreaterThan(0);
  });
});
