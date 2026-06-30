import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { PageBreadcrumb } from './PageBreadcrumb';

describe('PageBreadcrumb', () => {
  it('renders linked and current items', () => {
    render(
      <MemoryRouter>
        <PageBreadcrumb
          ariaLabel="Breadcrumb"
          items={[
            { label: 'Platform', href: '/shifts' },
            { label: 'Integrations' },
          ]}
        />
      </MemoryRouter>,
    );

    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Platform' })).toHaveAttribute('href', '/shifts');
    expect(screen.getByText('Integrations')).toBeInTheDocument();
  });
});
