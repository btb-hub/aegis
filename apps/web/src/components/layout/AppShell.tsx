import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { LanguageSwitcher } from './LanguageSwitcher';

type AppShellProps = {
  children: ReactNode;
};

export function AppShell({ children }: AppShellProps) {
  const { t } = useTranslation();

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-60 flex-col border-r border-zinc-200 bg-white">
        <div className="flex h-14 items-center border-b border-zinc-200 px-4 font-semibold">{t('app.title')}</div>
        <nav className="p-3 text-sm text-zinc-600">
          <div className="rounded-md bg-surface-muted px-3 py-2 font-medium text-zinc-900">{t('nav.home')}</div>
        </nav>
      </aside>
      <div className="flex min-h-screen flex-1 flex-col">
        <header className="flex h-14 items-center justify-end border-b border-zinc-200 bg-white px-4">
          <LanguageSwitcher />
        </header>
        <main className="flex-1 p-6">{children}</main>
      </div>
    </div>
  );
}
