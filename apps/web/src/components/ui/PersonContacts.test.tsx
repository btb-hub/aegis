import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import { PersonContacts } from './PersonContacts';

describe('PersonContacts', () => {
  it('renders all present contact links', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <PersonContacts
          displayName="Alice"
          contacts={{
            email: 'mailto:alice@example.com',
            slack: 'https://slack.com/app_redirect?channel=U123',
            express: 'https://xlnk.ms/open/profile/83fbf1c7-f14b-5176-bd32-ca15cf00d4b7',
          }}
        />
      </I18nextProvider>,
    );

    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Email' })).toHaveAttribute('href', 'mailto:alice@example.com');
    const slack = screen.getByRole('link', { name: 'Message in Slack' });
    expect(slack).toHaveAttribute('href', 'https://slack.com/app_redirect?channel=U123');
    expect(slack).toHaveAttribute('target', '_blank');
    expect(slack).toHaveAttribute('rel', 'noopener noreferrer');
    expect(screen.getByRole('link', { name: 'Message in eXpress' })).toHaveAttribute(
      'href',
      'https://xlnk.ms/open/profile/83fbf1c7-f14b-5176-bd32-ca15cf00d4b7',
    );
  });

  it('omits missing channels', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <PersonContacts displayName="Bob" contacts={{ email: 'mailto:bob@example.com' }} />
      </I18nextProvider>,
    );

    expect(screen.getByRole('link', { name: 'Email' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Message in Slack' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Message in eXpress' })).not.toBeInTheDocument();
  });
});
