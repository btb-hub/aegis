import { useTranslation } from 'react-i18next';
import { OnCallBanner } from '../components/shifts/OnCallBanner';
import { ShiftsCalendar } from '../components/shifts/ShiftsCalendar';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import type { CalendarOverride, CalendarSlot, OnCallUser } from '../lib/shiftsTypes';

type TeamShiftsPageProps = {
  teamName: string;
  onCallUsers: OnCallUser[];
  slots: CalendarSlot[];
  overrides: CalendarOverride[];
  month?: Date;
};

export function TeamShiftsPage({
  teamName,
  onCallUsers,
  slots,
  overrides,
  month = new Date(),
}: TeamShiftsPageProps) {
  const { t } = useTranslation();

  return (
    <PageContent>
      <PageHeader
        title={teamName}
        subtitle={t('shifts.page_subtitle')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [
            { label: t('nav.platform'), href: '/dashboard' },
            { label: t('nav.teams'), href: '/teams' },
            { label: teamName },
          ],
        }}
      />
      <OnCallBanner users={onCallUsers} />
      <ShiftsCalendar month={month} slots={slots} overrides={overrides} />
    </PageContent>
  );
}
