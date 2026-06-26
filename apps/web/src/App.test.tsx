import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it } from 'vitest';
import { App } from './App';
import i18n from './i18n';

describe('App', () => {
  it('renders translated sample content', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <App />
      </I18nextProvider>,
    );
    expect(screen.getByText('On-call schedule and overrides')).toBeInTheDocument();
    expect(screen.getByText(/on call now/i)).toBeInTheDocument();
    expect(screen.getAllByText('Bob').length).toBeGreaterThan(0);
  });
});
