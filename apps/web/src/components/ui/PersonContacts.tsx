import { useTranslation } from 'react-i18next';
import type { ContactLinks } from '../../lib/contactTypes';
import { Button } from './Button';

type PersonContactsProps = {
  displayName: string;
  contacts?: ContactLinks;
};

function isHttps(href: string): boolean {
  return href.startsWith('https:') || href.startsWith('http:');
}

export function PersonContacts({ displayName, contacts }: PersonContactsProps) {
  const { t } = useTranslation();
  const links = [
    contacts?.email ? { href: contacts.email, label: t('contacts.email') } : null,
    contacts?.slack ? { href: contacts.slack, label: t('contacts.slack') } : null,
    contacts?.express ? { href: contacts.express, label: t('contacts.express') } : null,
  ].filter((link): link is { href: string; label: string } => link !== null);

  return (
    <div>
      <p className="text-lg font-semibold text-zinc-900">{displayName}</p>
      {links.length > 0 ? (
        <div className="mt-1 flex flex-wrap gap-1">
          {links.map((link) => (
            <Button
              key={`${link.label}-${link.href}`}
              variant="ghost"
              href={link.href}
              target={isHttps(link.href) ? '_blank' : undefined}
              rel={isHttps(link.href) ? 'noopener noreferrer' : undefined}
            >
              {link.label}
            </Button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
