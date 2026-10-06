import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SettingsPage } from './SettingsPage';
import '../i18n';
const auth = vi.hoisted(() => ({ role: 'admin' }));
vi.mock('../context/AuthContext', () => ({ useAuth: () => ({ user: auth }) }));
const initial = { time: '03:00', timezone: 'UTC', next_run_at: '2026-10-06T03:00:00Z' };
beforeEach(() => { auth.role = 'admin'; vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ok:true,json:async()=>initial})); });
describe('publication settings', () => {
 it('loads the default and saves an IANA timezone without restarting',async()=> {
  render(<SettingsPage />);const time=await screen.findByLabelText(/Daily time/);
  expect(time).toHaveValue('03:00');expect(screen.getByText(/Next publication:/)).toHaveTextContent('UTC');
  fireEvent.change(time,{target:{value:'08:00'}});fireEvent.change(screen.getByLabelText(/Timezone/),{target:{value:'Europe/Moscow'}});
  fireEvent.click(screen.getByRole('button',{name:'Save'}));await screen.findByText('Settings saved.');
  expect(fetch).toHaveBeenLastCalledWith('/api/v1/settings/oncall-publication',expect.objectContaining({method:'PATCH',body:JSON.stringify({time:'08:00',timezone:'Europe/Moscow'})}));
 });
 it('handles invalid timezone editing and failed save',async()=> {
  render(<SettingsPage />);await screen.findByLabelText(/Timezone/);
  fireEvent.change(screen.getByLabelText(/Timezone/),{target:{value:'Invalid/'}});
  vi.mocked(fetch).mockResolvedValueOnce({ok:false} as Response);fireEvent.click(screen.getByRole('button',{name:'Save'}));await screen.findByText(/Could not load or save settings/);
 });
 it('shows a load error',async()=> {vi.mocked(fetch).mockRejectedValue(new Error('network'));render(<SettingsPage/>);await screen.findByText(/Could not load/);});
 it('does not load for members',async()=> {auth.role='member';render(<SettingsPage/>);expect(screen.getByText('Administrator access required.')).toBeInTheDocument();await waitFor(()=>expect(fetch).not.toHaveBeenCalled());});
});
