import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../i18n';
import { saveSetupWizardState } from '../lib/setupWizard';
import { SetupWizardPage } from './SetupWizardPage';

vi.mock('../context/AuthContext', () => ({
  useAuth: () => ({ user: { email: 'admin@example.com', display_name: 'Admin' } }),
}));

function renderWizard() {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <SetupWizardPage />
      </MemoryRouter>
    </I18nextProvider>,
  );
}

describe('SetupWizardPage', () => {
  beforeEach(() => {
    saveSetupWizardState({ step: 0, completed: false });
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo, init?: RequestInit) => {
        const url = String(input);
        if (url.includes('/healthz')) {
          return Promise.resolve({ ok: true });
        }
        if (url.includes('/integrations') && init?.method === 'POST') {
          return Promise.resolve({
            ok: true,
            json: async () => ({ id: 'int-1', kind: 'jira', name: 'Jira', enabled: true }),
          });
        }
        if (url.includes('/integrations/') && url.includes('/test')) {
          return Promise.resolve({ ok: true });
        }
        if (url.includes('/setup/test-alert')) {
          return Promise.resolve({ ok: true, json: async () => ({ id: 'alert-1' }) });
        }
        if (url.includes('/integrations')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({
              items: [{ id: 'int-1', kind: 'jira', name: 'Jira', enabled: true }],
            }),
          });
        }
        return Promise.resolve({ ok: true, json: async () => ({}) });
      }),
    );
  });

  it('renders wizard steps', async () => {
    renderWizard();
    expect(screen.getByText('Setup wizard')).toBeInTheDocument();
    expect(screen.getByText(/Welcome/)).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText(/API health check passed/)).toBeInTheDocument();
    });
  });

  it('navigates through integration and test alert steps', async () => {
    renderWizard();

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    expect(screen.getByText(/Signed in as/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    expect(screen.getByText('Configure connectors')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Jira base URL'), { target: { value: 'https://jira.example.com' } });
    fireEvent.change(screen.getByLabelText('Jira email'), { target: { value: 'ops@example.com' } });
    fireEvent.change(screen.getByLabelText('Jira API token'), { target: { value: 'token' } });
    fireEvent.change(screen.getByLabelText('Jira project key'), { target: { value: 'OPS' } });
    fireEvent.click(screen.getAllByRole('button', { name: 'Save connector' })[0]);

    await waitFor(() => {
      expect(screen.getByText(/jira saved/i)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
    fireEvent.change(screen.getByLabelText('Slack bot token'), { target: { value: 'xoxb-test' } });
    fireEvent.change(screen.getByLabelText('Slack signing secret'), { target: { value: 'secret' } });
    fireEvent.click(screen.getAllByRole('button', { name: 'Save connector' })[1]);
    fireEvent.change(screen.getByLabelText('eXpress bot ID'), { target: { value: 'bot' } });
    fireEvent.change(screen.getByLabelText('eXpress bot host'), { target: { value: 'https://express.example.com' } });
    fireEvent.change(screen.getByLabelText('eXpress secret key'), { target: { value: 'secret' } });
    fireEvent.click(screen.getAllByRole('button', { name: 'Save connector' })[2]);

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Send test alert' }));

    await waitFor(() => {
      expect(screen.getByText(/Alert accepted with id alert-1/)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    expect(screen.getByText('Setup complete')).toBeInTheDocument();
  });

  it('supports stepping back and retrying health check', async () => {
    const fetchMock = vi.fn().mockImplementation((input: RequestInfo) => {
      const url = String(input);
      if (url.includes('/healthz')) {
        return Promise.resolve({ ok: false });
      }
      if (url.includes('/integrations')) {
        return Promise.resolve({ ok: true, json: async () => ({ items: [] }) });
      }
      return Promise.resolve({ ok: true, json: async () => ({}) });
    });
    vi.stubGlobal('fetch', fetchMock);

    renderWizard();
    await waitFor(() => {
      expect(screen.getByText(/API health check failed/)).toBeInTheDocument();
    });

    fetchMock.mockImplementation((input: RequestInfo) => {
      if (String(input).includes('/healthz')) {
        return Promise.resolve({ ok: true });
      }
      return Promise.resolve({ ok: true, json: async () => ({ items: [] }) });
    });
    fireEvent.click(screen.getByRole('button', { name: 'Retry health check' }));
    await waitFor(() => {
      expect(screen.getByText(/API health check passed/)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByText(/API health check passed/)).toBeInTheDocument();
  });

  it('shows connector errors', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo, init?: RequestInit) => {
        const url = String(input);
        if (url.includes('/healthz')) {
          return Promise.resolve({ ok: true });
        }
        if (url.includes('/integrations') && init?.method === 'POST') {
          return Promise.resolve({ ok: false, json: async () => ({ message: 'save failed' }) });
        }
        if (url.includes('/integrations')) {
          return Promise.resolve({ ok: true, json: async () => ({ items: [] }) });
        }
        return Promise.resolve({ ok: true, json: async () => ({}) });
      }),
    );

    renderWizard();
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getAllByRole('button', { name: 'Save connector' })[0]);

    await waitFor(() => {
      expect(screen.getByText('save failed')).toBeInTheDocument();
    });
  });
});
