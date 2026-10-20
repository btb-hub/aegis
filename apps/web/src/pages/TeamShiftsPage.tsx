import type { ReactNode } from 'react';
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
  onDateChange?: (date: Date) => void;
  onAddOverride?: () => void;
  actions?: ReactNode;
  notice?: ReactNode;
};

export function TeamShiftsPage({
  teamName,
  onCallUsers,
  slots,
  overrides,
  month = new Date(),
  onDateChange,
  onAddOverride,
  actions,
  notice,
}: TeamShiftsPageProps) {
  const { t } = useTranslation();

  return (
    <PageContent>
      <PageHeader
        title={teamName}
        subtitle={t('shifts.page_subtitle')}
        actions={actions}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [
            { label: t('nav.platform'), href: '/dashboard' },
            { label: t('nav.teams'), href: '/teams' },
            { label: teamName },
          ],
        }}
      />
      {notice}
      <OnCallBanner users={onCallUsers} />
      <ShiftsCalendar month={month} teamName={teamName} slots={slots} overrides={overrides} onDateChange={onDateChange} onAddOverride={onAddOverride} />
    </PageContent>
  );
}
