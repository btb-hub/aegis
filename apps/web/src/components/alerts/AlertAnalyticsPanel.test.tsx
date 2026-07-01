import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import { AlertAnalyticsPanel } from './AlertAnalyticsPanel';

describe('AlertAnalyticsPanel', () => {
  it('renders nothing without analytics', () => {
    const { container } = render(
      <I18nextProvider i18n={i18n}>
        <AlertAnalyticsPanel analytics={null} />
      </I18nextProvider>,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it('renders severity, status, and top labels', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <AlertAnalyticsPanel
          analytics={{
            by_severity: { critical: 2 },
            by_status: { firing: 2 },
            top_labels: [
              { key: 'team', value: 'platform', count: 2 },
              { key: 'team', value: '', count: 1 },
            ],
          }}
        />
      </I18nextProvider>,
    );

    expect(screen.getByText('Inline analytics')).toBeInTheDocument();
    expect(screen.getByText('Critical')).toBeInTheDocument();
    expect(screen.getByText('team=platform')).toBeInTheDocument();
    expect(screen.getByText('team=(empty)')).toBeInTheDocument();
  });

  it('renders empty analytics sections', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <AlertAnalyticsPanel analytics={{ by_severity: {}, by_status: {}, top_labels: [] }} />
      </I18nextProvider>,
    );

    expect(screen.getAllByText('No data').length).toBeGreaterThanOrEqual(2);
  });
});
