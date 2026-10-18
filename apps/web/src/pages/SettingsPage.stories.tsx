import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../context/AuthContext';
import { SettingsPage } from './SettingsPage';
import { settingsStoryFixtures } from '../../.storybook/settingsFixtures';
const meta: Meta<typeof SettingsPage> = {
  title: 'Pages/SettingsPage', component: SettingsPage, tags: ['autodocs'],
  parameters: { layout: 'fullscreen' },
  beforeEach: settingsStoryFixtures,
  decorators: [(Story, context) => <MemoryRouter initialEntries={[context.parameters.settingsPath ?? '/settings']}><AuthProvider><Story /></AuthProvider></MemoryRouter>],
};
export default meta;
type Story = StoryObj<typeof SettingsPage>;
export const English: Story = { globals: { locale: 'en' } };
export const Russian: Story = { globals: { locale: 'ru' } };

export const AnomalyDetection: Story = { globals: { locale: 'en' }, parameters: { settingsPath: '/settings?section=anomaly' } };
export const AnomalyDetectionRussian: Story = { globals: { locale: 'ru' }, parameters: { settingsPath: '/settings?section=anomaly' } };
