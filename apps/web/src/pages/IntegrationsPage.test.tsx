import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../i18n';
import { IntegrationsPage } from './IntegrationsPage';

const slackIntegration = {
  id: 'int-slack',
  kind: 'slack',
  name: 'Slack',
  enabled: true,
};

const jiraIntegration = {
  id: 'int-jira',
  kind: 'jira',
  name: 'Jira',
  enabled: false,
};

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}

function renderPage() {
  return render(
    <MemoryRouter>
      <I18nextProvider i18n={i18n}>
        <IntegrationsPage />
      </I18nextProvider>
    </MemoryRouter>,
  );
}

function clickTestConnection(integrationName: string) {
  const row = screen.getByText(integrationName).closest('tr');
  expect(row).not.toBeNull();
  fireEvent.click(within(row as HTMLElement).getByRole('button', { name: 'Test connection' }));
}

describe('IntegrationsPage', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('shows breadcrumb navigation back to shifts', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ items: [] }));

    renderPage();

    expect(screen.getByRole('link', { name: 'Platform' })).toHaveAttribute('href', '/dashboard');
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Integrations');
    expect(screen.getByRole('heading', { name: 'Integrations', level: 1 })).toBeInTheDocument();
  });

  it('shows loading then empty state', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ items: [] }));

    renderPage();
    expect(screen.getByText('Loading integrations')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('No integrations configured yet')).toBeInTheDocument();
    });
  });

  it('renders integrations and tests connection successfully', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ items: [slackIntegration, jiraIntegration] }))
      .mockResolvedValueOnce(jsonResponse({}));

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack')).toBeInTheDocument();
    });
    expect(screen.getByText('Disabled')).toBeInTheDocument();

    clickTestConnection('Slack');

    await waitFor(() => {
      expect(screen.getByText('Connection succeeded')).toBeInTheDocument();
    });
    expect(fetch).toHaveBeenLastCalledWith('/api/v1/integrations/int-slack/test', {
      method: 'POST',
      credentials: 'include',
    });
  });

  it('shows sign-in message when load returns 401', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({}, 401));

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Your session expired. Sign in again.');
    });
  });

  it('shows load error when fetch fails', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({}, 500));

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Could not load integrations');
    });
  });

  it('shows load error on network failure', async () => {
    vi.mocked(fetch).mockRejectedValueOnce(new Error('network'));

    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Could not load integrations');
    });
  });

  it('shows sign-in toast when test returns 401', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ items: [slackIntegration] }))
      .mockResolvedValueOnce(jsonResponse({}, 401));

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack')).toBeInTheDocument();
    });
    clickTestConnection('Slack');

    await waitFor(() => {
      expect(screen.getByText('Your session expired. Sign in again.')).toBeInTheDocument();
    });
  });

  it('shows API error message when test fails', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ items: [slackIntegration] }))
      .mockResolvedValueOnce(jsonResponse({ message: 'Invalid token' }, 400));

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack')).toBeInTheDocument();
    });
    clickTestConnection('Slack');

    await waitFor(() => {
      expect(screen.getByText('Invalid token')).toBeInTheDocument();
    });
  });

  it('shows testing label while connection test is in flight', async () => {
    let resolveTest!: (value: Response) => void;
    const testPromise = new Promise<Response>((resolve) => {
      resolveTest = resolve;
    });

    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ items: [slackIntegration] }))
      .mockReturnValueOnce(testPromise);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack')).toBeInTheDocument();
    });
    clickTestConnection('Slack');

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Testing' })).toBeDisabled();
    });

    resolveTest(jsonResponse({}));
    await waitFor(() => {
      expect(screen.getByText('Connection succeeded')).toBeInTheDocument();
    });
  });

  it('shows generic test failure on unexpected rejection', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ items: [slackIntegration] }))
      .mockRejectedValueOnce('network');

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Slack')).toBeInTheDocument();
    });
    clickTestConnection('Slack');

    await waitFor(() => {
      expect(screen.getByText('Connection failed')).toBeInTheDocument();
    });
  });
});
