export type OnCallUser = {
  userId: string;
  displayName: string;
  email: string;
  source: 'rotation' | 'override';
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
