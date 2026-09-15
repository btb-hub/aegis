export type ContactLinks = {
  email?: string;
  slack?: string;
  express?: string;
};

export type ContactPerson = {
  userId: string;
  email: string;
  displayName: string;
  contacts?: ContactLinks;
};
