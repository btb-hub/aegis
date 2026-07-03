import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import type { UserDirectoryItem } from '../../lib/teamTypes';
import { TeamMemberPicker } from './TeamMemberPicker';

const sampleUsers: UserDirectoryItem[] = [
  { id: 'user-1', email: 'alice@example.com', display_name: 'Alice', role: 'member' },
  { id: 'user-2', email: 'bob@example.com', display_name: 'Bob', role: 'member' },
];

const meta: Meta<typeof TeamMemberPicker> = {
  title: 'Teams/TeamMemberPicker',
  component: TeamMemberPicker,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof TeamMemberPicker>;

function PickerDemo() {
  const [selected, setSelected] = useState<UserDirectoryItem | null>(null);

  return (
    <div className="max-w-md space-y-4">
      <TeamMemberPicker onSelect={setSelected} excludeUserIds={['user-1']} />
      {selected ? (
        <p className="text-sm text-zinc-700">
          Selected: {selected.display_name} ({selected.email})
        </p>
      ) : null}
    </div>
  );
}

export const English: Story = {
  globals: { locale: 'en' },
  render: () => <PickerDemo />,
  beforeEach: () => {
    globalThis.fetch = (async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/api/v1/users')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({ items: sampleUsers }),
        } as Response;
      }
      return { ok: false, status: 404, json: async () => ({}) } as Response;
    }) as typeof fetch;
  },
};

export const Russian: Story = {
  globals: { locale: 'ru' },
  render: () => <PickerDemo />,
  beforeEach: () => {
    globalThis.fetch = (async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/api/v1/users')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({ items: sampleUsers }),
        } as Response;
      }
      return { ok: false, status: 404, json: async () => ({}) } as Response;
    }) as typeof fetch;
  },
};
