import type { ContactLinks } from './contactTypes';

export type OnCallUser = {
  userId: string;
  displayName: string;
  email: string;
  source: 'rotation' | 'override';
  contacts?: ContactLinks;
};

export type CalendarSlot = {
  id: string;
  userId: string;
  displayName: string;
  startAt: string;
  endAt: string;
  source: 'rotation' | 'override';
};

export type CalendarOverride = {
  id: string;
  userId: string;
  displayName: string;
  startAt: string;
  endAt: string;
};
