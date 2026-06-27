import { useTranslation } from 'react-i18next';
import { AppShell } from './components/layout/AppShell';
import { TeamShiftsPage } from './pages/TeamShiftsPage';

const demoSlots = [
  {
    id: 'slot-1',
    userId: 'user-alice',
    displayName: 'Alice',
    startAt: '2026-06-02T09:00:00Z',
    endAt: '2026-06-09T09:00:00Z',
    source: 'rotation' as const,
  },
  {
    id: 'slot-2',
    userId: 'user-bob',
    displayName: 'Bob',
    startAt: '2026-06-09T09:00:00Z',
    endAt: '2026-06-16T09:00:00Z',
    source: 'rotation' as const,
  },
];

const demoOverrides = [
  {
    id: 'override-1',
    userId: 'user-carol',
    displayName: 'Carol',
    startAt: '2026-06-12T00:00:00Z',
    endAt: '2026-06-13T00:00:00Z',
  },
];

export function App() {
  const { t } = useTranslation();

  return (
    <AppShell>
      <TeamShiftsPage
        teamName={t('shifts.demo_team')}
        onCallUsers={[{ userId: 'user-bob', displayName: 'Bob', email: 'bob@example.com', source: 'rotation' }]}
        slots={demoSlots}
        overrides={demoOverrides}
        month={new Date('2026-06-01T00:00:00Z')}
      />
    </AppShell>
  );
}
