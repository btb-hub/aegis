import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';
import { Banner } from '../components/ui/Banner';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';

type PublicationSettings = { time: string; timezone: string; next_run_at: string };
export function SettingsPage() {
  const { t, i18n } = useTranslation();
  const { user } = useAuth();
  const [settings, setSettings] = useState<PublicationSettings | null>(null);
  const [error, setError] = useState(false);
  const [saved, setSaved] = useState(false);
  const [saving, setSaving] = useState(false);
  useEffect(() => {
    if (user?.role !== 'admin') {return;}
    void fetch('/api/v1/settings/oncall-publication', { credentials: 'include' })
      .then(async response => {
        if (!response.ok) {throw new Error('settings');}
        setSettings(await response.json() as PublicationSettings);
      }).catch(() => setError(true));
  }, [user?.role]);
  async function save() {
    setSaving(true); setError(false); setSaved(false);
    try {
      const response = await fetch('/api/v1/settings/oncall-publication', {
        method: 'PATCH', credentials: 'include', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ time: settings?.time, timezone: settings?.timezone }),
      });
      if (!response.ok) {throw new Error('settings');}
      setSettings(await response.json() as PublicationSettings); setSaved(true);
    } catch { setError(true); } finally { setSaving(false); }
  }
  let nextTime = settings?.next_run_at ?? '';
  try { if (settings) {nextTime = new Intl.DateTimeFormat(i18n.language, { dateStyle: 'medium', timeStyle: 'short', timeZone: settings.timezone }).format(new Date(settings.next_run_at));} } catch { /* Keep saved instant visible while editing an incomplete timezone. */ }
  if (user?.role !== 'admin') {return <Banner variant="warning">{t('settings.admin_required')}</Banner>;}
  return <PageContent>
    <PageHeader title={t('nav.settings')} subtitle={t('settings.description')} />
    {error && <Banner variant="warning">{t('settings.error')}</Banner>}
    {saved && <p role="status">{t('settings.saved')}</p>}
    {settings && <section className="max-w-lg space-y-4 rounded-md border border-zinc-200 bg-white p-4">
      <h2 className="font-semibold">{t('settings.publication')}</h2>
      <Input label={t('settings.time')} value={settings.time} hint="HH:mm" onChange={time => setSettings({ ...settings, time })} />
      <Input label={t('settings.timezone')} value={settings.timezone} hint={t('settings.timezone_hint')} onChange={timezone => setSettings({ ...settings, timezone })} />
      <p>{t('settings.next', { time: nextTime, timezone: settings.timezone })}</p>
      <Button disabled={saving} onClick={() => void save()}>{t('actions.save')}</Button>
    </section>}
  </PageContent>;
}
