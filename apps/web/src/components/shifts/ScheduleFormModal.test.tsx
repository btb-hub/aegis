import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { ScheduleFormModal } from './ScheduleFormModal';

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
  {
    id: 'm2',
    team_id: 'team-1',
    user_id: 'u2',
    team_role: 'member' as const,
    email: 'b@example.com',
    display_name: 'Bob',
    created_at: '',
  },
];

describe('ScheduleFormModal', () => {
  it('requires at least one participant', async () => {
    const onSave = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <ScheduleFormModal open members={members} onClose={() => undefined} onSave={onSave} />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Save schedule' }));

    await waitFor(() => {
      expect(screen.getByText('Select at least one team member')).toBeInTheDocument();
    });
    expect(onSave).not.toHaveBeenCalled();
  });

  it('submits schedule when participant selected', async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(
      <I18nextProvider i18n={i18n}>
        <ScheduleFormModal open members={members} onClose={() => undefined} onSave={onSave} />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByRole('checkbox', { name: 'Alice' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save schedule' }));

    await waitFor(() => {
      expect(onSave).toHaveBeenCalled();
    });
  });

  it('reorders selected participants', async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(
      <I18nextProvider i18n={i18n}>
        <ScheduleFormModal open members={members} onClose={() => undefined} onSave={onSave} />
      </I18nextProvider>,
    );

    fireEvent.click(screen.getByRole('checkbox', { name: 'Alice' }));
    fireEvent.click(screen.getByRole('checkbox', { name: 'Bob' }));
    expect(screen.getByRole('button', { name: 'Move Alice earlier' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: 'Move Alice later' }));
    expect(screen.getAllByRole('checkbox')[0]).toHaveAccessibleName('Bob');
    fireEvent.click(screen.getByRole('button', { name: 'Save schedule' }));

    await waitFor(() => {
      expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ participants: ['u2', 'u1'] }));
    });
  });
});
