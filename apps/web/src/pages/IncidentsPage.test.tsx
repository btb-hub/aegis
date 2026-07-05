import { fireEvent, render, screen } from '@testing-library/react';
import type { ComponentProps } from 'react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { IncidentsPage } from './IncidentsPage';
import i18n from '../i18n';
import type { Incident } from '../lib/incidentTypes';

const incidents: Incident[] = [
  {
    id: '11111111-1111-1111-1111-111111111111',
    teamId: 'team-1',
    status: 'open',
    severity: 'critical',
    title: 'CPU high',
    fingerprint: 'fp-1',
    createdAt: '2026-06-26T10:00:00Z',
    alerts: [],
    timeline: [],
  },
];

describe('IncidentsPage', () => {
  const renderPage = (props: ComponentProps<typeof IncidentsPage>) =>
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <IncidentsPage {...props} />
        </MemoryRouter>
      </I18nextProvider>,
    );

  it('renders list and detail together', () => {
    renderPage({
      incidents,
      handoffTeams: [{ id: 'team-l3', name: 'L3' }],
      onAcknowledge: vi.fn(),
      onResolve: vi.fn(),
      onHandoff: vi.fn(),
      onBounce: vi.fn(),
    });

    expect(screen.getByRole('heading', { name: 'Incidents' })).toBeInTheDocument();
    expect(screen.getAllByText('CPU high').length).toBeGreaterThan(0);
  });

  it('filters incidents from the page controls', () => {
    renderPage({
      incidents: [
        ...incidents,
        {
          ...incidents[0],
          id: '22222222-2222-2222-2222-222222222222',
          title: 'Resolved item',
          status: 'resolved',
        },
      ],
      handoffTeams: [{ id: 'team-l3', name: 'L3' }],
      onAcknowledge: vi.fn(),
      onResolve: vi.fn(),
      onHandoff: vi.fn(),
      onBounce: vi.fn(),
    });

    fireEvent.click(screen.getByRole('button', { name: 'Open' }));
    expect(screen.getByRole('button', { name: /CPU high/i })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Resolved item/i })).not.toBeInTheDocument();
  });

  it('shows a prompt when no incident is selected', () => {
    renderPage({
      incidents: [],
      handoffTeams: [],
      onAcknowledge: vi.fn(),
      onResolve: vi.fn(),
      onHandoff: vi.fn(),
      onBounce: vi.fn(),
    });

    expect(screen.getByText('Select an incident to view details')).toBeInTheDocument();
  });
});
