import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { PageHeader } from './PageHeader';

describe('PageHeader', () => {
  it('renders title, subtitle, and actions', () => {
    render(
      <MemoryRouter>
        <PageHeader
          title="Alerts"
          subtitle="Search and filter alert history"
          breadcrumb={{
            ariaLabel: 'Breadcrumb',
            items: [{ label: 'Platform', href: '/dashboard' }, { label: 'Alerts' }],
          }}
          actions={<button type="button">Export CSV</button>}
        />
      </MemoryRouter>,
    );
    expect(screen.getByRole('heading', { name: 'Alerts' })).toBeInTheDocument();
    expect(screen.getByText('Search and filter alert history')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Export CSV' })).toBeInTheDocument();
  });
});
