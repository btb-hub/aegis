import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import { OnCallBanner } from './OnCallBanner';

describe('OnCallBanner', () => {
  it('shows on-call names', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <OnCallBanner
          users={[{ userId: '1', displayName: 'Alice', email: 'a@example.com', source: 'rotation' }]}
        />
      </I18nextProvider>,
    );
    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByText(/on call now/i)).toBeInTheDocument();
  });

  it('shows empty state', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <OnCallBanner users={[]} />
      </I18nextProvider>,
    );
    expect(screen.getByText(/no one is on call/i)).toBeInTheDocument();
  });
});
