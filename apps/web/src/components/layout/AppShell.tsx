import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { LanguageSwitcher } from './LanguageSwitcher';

export type AppPage = 'shifts' | 'incidents';

type AppShellProps = {
  children: ReactNode;
  currentPage?: AppPage;
  onNavigate?: (page: AppPage) => void;
};

export function AppShell({ children, currentPage = 'shifts', onNavigate }: AppShellProps) {
  const { t } = useTranslation();

  const navItems: Array<{ id: AppPage; label: string }> = [
    { id: 'shifts', label: t('nav.shifts') },
    { id: 'incidents', label: t('nav.incidents') },
  ];

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-60 flex-col border-r border-zinc-200 bg-white">
        <div className="flex h-14 items-center border-b border-zinc-200 px-4 font-semibold">{t('app.title')}</div>
        <nav className="space-y-1 p-3 text-sm text-zinc-600">
          {navItems.map((item) => {
            const active = currentPage === item.id;
            if (onNavigate) {
              return (
                <button
                  key={item.id}
                  type="button"
                  className={`block w-full rounded-md px-3 py-2 text-left font-medium ${
                    active ? 'bg-surface-muted text-zinc-900' : 'text-zinc-600 hover:bg-zinc-50'
                  }`}
                  onClick={() => onNavigate(item.id)}
                >
                  {item.label}
                </button>
              );
            }
            return (
              <div
                key={item.id}
                className={`rounded-md px-3 py-2 font-medium ${active ? 'bg-surface-muted text-zinc-900' : ''}`}
              >
                {item.label}
              </div>
            );
          })}
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
