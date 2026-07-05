import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useNavigate } from 'react-router-dom';
import { LanguageSwitcher } from './LanguageSwitcher';
import type { AuthUser } from '../../lib/authTypes';
import { Button } from '../ui/Button';

export type AppPage = 'shifts' | 'teams' | 'incidents' | 'alerts' | 'integrations' | 'dashboard' | 'setup';

type AppShellProps = {
  children: ReactNode;
  currentPage?: AppPage;
  onNavigate?: (page: AppPage) => void;
  user?: AuthUser | null;
  onSignOut?: () => void | Promise<void>;
};

export function AppShell({ children, currentPage = 'shifts', onNavigate, user, onSignOut }: AppShellProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const navItems: Array<{ id: AppPage; label: string }> = [
    { id: 'shifts', label: t('nav.shifts') },
    { id: 'teams', label: t('nav.teams') },
    { id: 'incidents', label: t('nav.incidents') },
    { id: 'alerts', label: t('nav.alerts') },
    { id: 'dashboard', label: t('nav.dashboard') },
    { id: 'integrations', label: t('nav.integrations') },
    { id: 'setup', label: t('nav.setup') },
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
        <header className="flex h-14 items-center justify-end gap-4 border-b border-zinc-200 bg-white px-4">
          {user ? (
            <div className="flex items-center gap-3 text-sm text-zinc-700">
              <Link to="/account" className="font-medium text-zinc-900 hover:text-accent">
                {user.display_name || user.email}
              </Link>
              <Button
                variant="ghost"
                onClick={() => {
                  void (async () => {
                    await onSignOut?.();
                    navigate('/login');
                  })();
                }}
              >
                {t('auth.sign_out')}
              </Button>
            </div>
          ) : null}
          <LanguageSwitcher />
        </header>
        <main className="flex-1 p-6">
          <div className="mx-auto w-full max-w-7xl">{children}</div>
        </main>
      </div>
    </div>
  );
}
