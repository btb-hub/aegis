import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Pagination } from './Pagination';

describe('Pagination', () => {
  it('navigates pages', () => {
    const onPageChange = vi.fn();
    render(
      <Pagination
        page={2}
        pageSize={25}
        total={75}
        onPageChange={onPageChange}
        totalLabel="75 alerts"
        prevLabel="Previous"
        nextLabel="Next"
        pageLabel="Page 2 of 3"
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Previous' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    expect(onPageChange).toHaveBeenCalledWith(1);
    expect(onPageChange).toHaveBeenCalledWith(3);
  });
});
