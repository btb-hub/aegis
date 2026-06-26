import { useTranslation } from 'react-i18next';
import { AppShell } from './components/layout/AppShell';
import { Button } from './components/ui/Button';
import { SeverityTag } from './components/ui/SeverityTag';
import { formatDateTime } from './lib/formatDate';

export function App() {
  const { t, i18n } = useTranslation();
  const sampleDate = formatDateTime(new Date('2026-06-26T12:00:00Z'), i18n.language);

  return (
    <AppShell>
      <div className="max-w-3xl space-y-6">
        <div>
          <h1 className="text-3xl font-semibold">{t('app.title')}</h1>
          <p className="text-zinc-600">{t('app.tagline')}</p>
        </div>
        <div className="flex items-center gap-3">
          <SeverityTag severity="P1" />
          <Button>{t('sample.primary')}</Button>
        </div>
        <p className="font-mono text-sm text-zinc-600">
          {t('sample.date_label')}: {sampleDate}
        </p>
      </div>
    </AppShell>
  );
}
