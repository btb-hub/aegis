import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import type { TeamMember } from '../../lib/teamTypes';
import { OverrideFormModal } from './OverrideFormModal';

const members: TeamMember[] = [
  {
    id: 'm1',
    team_id: 'team-1',
    user_id: 'u1',
    team_role: 'member',
    email: 'alice@example.com',
    display_name: 'Alice',
    created_at: '',
  },
];

const meta: Meta<typeof OverrideFormModal> = {
  title: 'Shifts/OverrideFormModal',
  component: OverrideFormModal,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof OverrideFormModal>;

function Demo() {
  const [open, setOpen] = useState(true);
  return (
    <OverrideFormModal
      open={open}
      onClose={() => setOpen(false)}
      members={members}
      overrides={[]}
      nameByUserId={new Map([['u1', 'Alice']])}
      onCreate={async () => undefined}
      onDelete={async () => undefined}
    />
  );
}

export const English: Story = {
  render: () => <Demo />,
  globals: { locale: 'en' },
};

export const Russian: Story = {
  render: () => <Demo />,
  globals: { locale: 'ru' },
};
