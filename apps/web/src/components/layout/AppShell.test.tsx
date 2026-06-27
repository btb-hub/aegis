import { fireEvent, render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { AppShell } from './AppShell';

describe('AppShell', () => {
  it('renders shell chrome', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <AppShell>
          <div>content</div>
        </AppShell>
      </I18nextProvider>,
    );
    expect(screen.getByText('content')).toBeInTheDocument();
    expect(screen.getByText('Shifts')).toBeInTheDocument();
  });

  it('renders Russian navigation when locale is ru', async () => {
    await i18n.changeLanguage('ru');
    render(
      <I18nextProvider i18n={i18n}>
        <AppShell>
          <div>x</div>
        </AppShell>
      </I18nextProvider>,
    );
    expect(screen.getByText('Смены')).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });

  it('calls onNavigate when a nav item is clicked', () => {
    const onNavigate = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <AppShell currentPage="shifts" onNavigate={onNavigate}>
          <div>content</div>
        </AppShell>
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Incidents' }));
    expect(onNavigate).toHaveBeenCalledWith('incidents');
  });
});
