import type { Meta, StoryObj } from '@storybook/react';
import { PersonContacts } from './PersonContacts';

const meta: Meta<typeof PersonContacts> = {
  title: 'UI/PersonContacts',
  component: PersonContacts,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof PersonContacts>;

export const AllChannels: Story = {
  args: {
    displayName: 'Alice Kim',
    contacts: {
      email: 'mailto:alice@example.com',
      slack: 'https://slack.com/app_redirect?channel=U123',
      express: 'https://xlnk.ms/open/profile/83fbf1c7-f14b-5176-bd32-ca15cf00d4b7',
    },
  },
  globals: { locale: 'en' },
};

export const AllChannelsRussian: Story = {
  args: {
    displayName: 'Алиса Ким',
    contacts: {
      email: 'mailto:alice@example.com',
      slack: 'https://slack.com/app_redirect?channel=U123',
      express: 'https://xlnk.ms/open/profile/83fbf1c7-f14b-5176-bd32-ca15cf00d4b7',
    },
  },
  globals: { locale: 'ru' },
};

export const EmailOnly: Story = {
  args: {
    displayName: 'Carol Diaz',
    contacts: { email: 'mailto:carol@example.com' },
  },
  globals: { locale: 'en' },
};
