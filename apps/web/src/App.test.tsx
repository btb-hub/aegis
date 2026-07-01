import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { App } from './App';
import { AuthProvider } from './context/AuthContext';
import i18n from './i18n';

function renderApp(initialPath = '/shifts') {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter initialEntries={[initialPath]}>
        <AuthProvider>
          <App />
        </AuthProvider>
      </MemoryRouter>
    </I18nextProvider>,
  );
}

describe('App', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-06-10T12:00:00Z'));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it('renders translated sample content', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({}),
    } as Response);

    renderApp('/shifts');

    await waitFor(() => {
      expect(screen.getByText('On-call schedule and overrides')).toBeInTheDocument();
    });
    expect(screen.getByText(/on call now/i)).toBeInTheDocument();
    expect(screen.getAllByText('Bob').length).toBeGreaterThan(0);
    expect(screen.getByText('Shifts')).toBeInTheDocument();
    expect(screen.getByText('Incidents')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Incidents'));
    expect(screen.getByText('Track open incidents, linked alerts, and timeline events')).toBeInTheDocument();
  });

  it('navigates to alerts workspace when signed in', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo) => {
      const url = String(input);
      if (url.includes('/auth/me')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({
            id: 'user-1',
            email: 'alice@example.com',
            display_name: 'Alice',
            role: 'admin',
            locale: 'en',
            provider: 'google',
          }),
        } as Response;
      }
      if (url.includes('/saved-views')) {
        return { ok: true, status: 200, json: async () => ({ items: [] }) } as Response;
      }
      if (url.includes('/alerts')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({ items: [], total: 0, analytics: { by_severity: {}, by_status: {}, top_labels: [] } }),
        } as Response;
      }
      return { ok: false, status: 401, json: async () => ({}) } as Response;
    });

    renderApp('/shifts');
    await waitFor(() => {
      expect(screen.getAllByText('Alice').length).toBeGreaterThan(0);
    });

    fireEvent.click(screen.getByText('Alerts'));
    expect(await screen.findByText('Search, filter, group, and export alert history')).toBeInTheDocument();
  });

  it('acknowledges and resolves incidents from the demo list', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({}),
    } as Response);

    renderApp('/incidents');

    await waitFor(() => {
      expect(screen.getAllByText('CPU high on api-1').length).toBeGreaterThan(0);
    });

    fireEvent.click(screen.getByRole('button', { name: 'Acknowledge' }));
    await waitFor(() => {
      expect(screen.getAllByText('Acknowledged').length).toBeGreaterThan(0);
    });

    fireEvent.click(screen.getByRole('button', { name: 'Resolve' }));
    await waitFor(() => {
      expect(screen.getAllByText('Resolved').length).toBeGreaterThan(0);
    });
  });

  it('redirects unsigned users from integrations to login', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({}),
    } as Response);

    renderApp('/integrations');

    expect(await screen.findByRole('link', { name: 'Sign in with Google' })).toBeInTheDocument();
  });
});
