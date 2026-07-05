import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { Checkbox } from './Checkbox';

const meta: Meta<typeof Checkbox> = {
  title: 'UI/Checkbox',
  component: Checkbox,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Checkbox>;

function CheckboxDemo() {
  const [checked, setChecked] = useState(false);
  return <Checkbox label="Share with team" checked={checked} onChange={setChecked} />;
}

export const Default: Story = {
  render: () => <CheckboxDemo />,
};

export const Checked: Story = {
  render: () => <Checkbox label="Share with team" checked onChange={() => undefined} />,
};
