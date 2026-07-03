import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { OverrideFormModal } from './OverrideFormModal';

const members = [
  {
    id: 'm1',
    team_id: 'team-1',
    user_id: 'u1',
    team_role: 'member' as const,
    email: 'a@example.com',
    display_name: 'Alice',
    created_at: '',
  },
];

describe('OverrideFormModal', () => {
  it('validates end after start', async () => {
    const onCreate = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <OverrideFormModal
          open
          onClose={() => undefined}
          members={members}
          overrides={[]}
          nameByUserId={new Map([['u1', 'Alice']])}
          onCreate={onCreate}
          onDelete={async () => undefined}
        />
      </I18nextProvider>,
    );

    fireEvent.change(screen.getByLabelText('Start'), { target: { value: '2026-06-10T12:00' } });
    fireEvent.change(screen.getByLabelText('End'), { target: { value: '2026-06-10T10:00' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save override' }));

    await waitFor(() => {
      expect(screen.getByText('End must be after start')).toBeInTheDocument();
    });
    expect(onCreate).not.toHaveBeenCalled();
  });

  it('creates override with valid range', async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined);
    render(
      <I18nextProvider i18n={i18n}>
        <OverrideFormModal
          open
          onClose={() => undefined}
          members={members}
          overrides={[]}
          nameByUserId={new Map([['u1', 'Alice']])}
          onCreate={onCreate}
          onDelete={async () => undefined}
        />
      </I18nextProvider>,
    );

    fireEvent.change(screen.getByLabelText('Start'), { target: { value: '2026-06-10T08:00' } });
    fireEvent.change(screen.getByLabelText('End'), { target: { value: '2026-06-10T16:00' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save override' }));

    await waitFor(() => {
      expect(onCreate).toHaveBeenCalled();
    });
  });

  it('deletes listed override', async () => {
    const onDelete = vi.fn().mockResolvedValue(undefined);
    render(
      <I18nextProvider i18n={i18n}>
        <OverrideFormModal
          open
          onClose={() => undefined}
          members={members}
          overrides={[
            {
              id: 'o1',
              team_id: 'team-1',
              user_id: 'u1',
              start_at: '2026-06-10T08:00:00Z',
              end_at: '2026-06-10T16:00:00Z',
            },
          ]}
          nameByUserId={new Map([['u1', 'Alice']])}
          onCreate={async () => undefined}
          onDelete={onDelete}
        />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Delete team' }));
    await waitFor(() => {
      expect(onDelete).toHaveBeenCalledWith('o1');
    });
  });
});
