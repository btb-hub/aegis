import { useTranslation } from 'react-i18next';
import { OnCallBanner } from '../components/shifts/OnCallBanner';
import { ShiftsCalendar } from '../components/shifts/ShiftsCalendar';
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
    <div className="max-w-5xl space-y-6">
      <div>
        <h1 className="text-3xl font-semibold">{teamName}</h1>
        <p className="text-zinc-600">{t('shifts.page_subtitle')}</p>
      </div>
      <OnCallBanner users={onCallUsers} />
      <ShiftsCalendar month={month} slots={slots} overrides={overrides} />
    </div>
  );
}
