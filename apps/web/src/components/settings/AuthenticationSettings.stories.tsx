import type { Meta, StoryObj } from '@storybook/react';
import { AuthenticationSettings } from './AuthenticationSettings';
import { settingsFixture, settingsStoryFixtures } from '../../../.storybook/settingsFixtures';
const meta: Meta<typeof AuthenticationSettings> = {
  title: 'Settings/Authentication', component: AuthenticationSettings, tags: ['autodocs'],
  beforeEach: settingsStoryFixtures,
  args: { settings: settingsFixture, email: 'admin@example.com' },
};
export default meta;
type Story = StoryObj<typeof AuthenticationSettings>;
export const English: Story = { globals: { locale: 'en' } };
export const Russian: Story = { globals: { locale: 'ru' } };
export const FirstInstallation: Story = { args: { bootstrap: true } };
