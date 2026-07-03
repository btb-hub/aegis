import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { Input } from './Input';
import { Modal } from './Modal';

const meta: Meta<typeof Modal> = {
  title: 'UI/Modal',
  component: Modal,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Modal>;

function ModalDemo() {
  const [open, setOpen] = useState(true);
  const [name, setName] = useState('Platform');

  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>
        Open modal
      </button>
      <Modal
        title="Create team"
        open={open}
        onClose={() => setOpen(false)}
        primaryLabel="Save"
        secondaryLabel="Cancel"
        onPrimary={() => setOpen(false)}
      >
        <Input label="Name" value={name} onChange={setName} />
      </Modal>
    </>
  );
}

export const Default: Story = {
  render: () => <ModalDemo />,
};
