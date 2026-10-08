import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../context/AuthContext';
import { SettingsPage } from './SettingsPage';
import { settingsStoryFixtures } from '../../.storybook/settingsFixtures';
const meta: Meta<typeof SettingsPage> = {
  title: 'Pages/SettingsPage', component: SettingsPage, tags: ['autodocs'],
  parameters: { layout: 'fullscreen' },
  beforeEach: settingsStoryFixtures,
  decorators: [(Story) => <MemoryRouter initialEntries={['/settings']}><AuthProvider><Story /></AuthProvider></MemoryRouter>],
};
export default meta;
type Story = StoryObj<typeof SettingsPage>;
export const English: Story = { globals: { locale: 'en' } };
export const Russian: Story = { globals: { locale: 'ru' } };
