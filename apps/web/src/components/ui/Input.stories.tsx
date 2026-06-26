import type { ComponentProps } from 'react';
import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { Input } from './Input';

const meta: Meta<typeof Input> = {
  title: 'UI/Input',
  component: Input,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Input>;

function InputDemo(props: Omit<ComponentProps<typeof Input>, 'value' | 'onChange'>) {
  const [value, setValue] = useState('on-call rotation');
  return <Input {...props} value={value} onChange={setValue} />;
}

export const Default: Story = {
  render: () => <InputDemo label="Team name" />,
};

export const Focused: Story = {
  render: () => <InputDemo label="Team name" />,
  parameters: { pseudo: { focus: true } },
};

export const WithError: Story = {
  render: () => <InputDemo label="Team name" error="Team name is required" />,
};
