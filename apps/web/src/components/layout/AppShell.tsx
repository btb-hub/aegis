import { useEffect, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useNavigate } from 'react-router-dom';
import { LanguageSwitcher } from './LanguageSwitcher';
import { AppVersion } from './AppVersion';
import type { AuthUser } from '../../lib/authTypes';
import { Button } from '../ui/Button';

export type AppPage =
  | 'shifts'
  | 'teams'
  | 'workspaces'
  | 'incidents'
  | 'alerts'
  | 'integrations'
  | 'dashboard'
  | 'setup'
  | 'users';

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
  const [navigationOpen, setNavigationOpen] = useState(false);

  const isAdmin = user?.role === 'admin';

  const navItems: Array<{ id: AppPage; label: string }> = [
    { id: 'shifts', label: t('nav.shifts') },
    { id: 'teams', label: t('nav.teams') },
    { id: 'workspaces', label: t('nav.workspaces') },
    { id: 'incidents', label: t('nav.incidents') },
    { id: 'alerts', label: t('nav.alerts') },
    { id: 'dashboard', label: t('nav.dashboard') },
    { id: 'integrations', label: t('nav.integrations') },
    { id: 'setup', label: t('nav.setup') },
    ...(isAdmin ? [{ id: 'users' as AppPage, label: t('nav.users') }] : []),
  ];

  useEffect(() => {
    setNavigationOpen(false);
  }, [currentPage]);

  useEffect(() => {
    if (!navigationOpen) {
      return undefined;
    }

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setNavigationOpen(false);
      }
    };

    window.addEventListener('keydown', closeOnEscape);
    return () => window.removeEventListener('keydown', closeOnEscape);
  }, [navigationOpen]);

  const renderNavigationItems = (closeAfterNavigation: boolean) =>
    navItems.map((item) => {
      const active = currentPage === item.id;
      if (onNavigate) {
        return (
          <button
            key={item.id}
            type="button"
            className={`block w-full rounded-md px-3 py-2 text-left font-medium ${
              active ? 'bg-surface-muted text-zinc-900' : 'text-zinc-600 hover:bg-zinc-50'
            }`}
            aria-current={active ? 'page' : undefined}
            onClick={() => {
              if (closeAfterNavigation) {
                setNavigationOpen(false);
              }
              onNavigate(item.id);
            }}
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
    });

  return (
    <div className="min-h-screen bg-zinc-50 lg:flex">
      <aside className="hidden w-60 shrink-0 flex-col border-r border-zinc-200 bg-white lg:flex">
        <div className="flex h-14 items-center border-b border-zinc-200 px-4 font-semibold">
          {t('app.title')}
        </div>
        <nav className="space-y-1 p-3 text-sm text-zinc-600">{renderNavigationItems(false)}</nav>
      </aside>
      {navigationOpen ? (
        <>
          <button
            type="button"
            className="fixed inset-0 z-30 bg-zinc-950/30 lg:hidden"
            aria-label={t('nav.close_menu')}
            onClick={() => setNavigationOpen(false)}
          />
          <aside
            id="primary-navigation"
            className="fixed inset-y-0 left-0 z-40 flex w-60 flex-col border-r border-zinc-200 bg-white shadow-xl lg:hidden"
          >
            <div className="flex h-14 items-center justify-between border-b border-zinc-200 px-4 font-semibold">
              <span>{t('app.title')}</span>
              <button
                type="button"
                className="inline-flex h-9 w-9 items-center justify-center rounded-md text-zinc-600 hover:bg-zinc-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
                aria-label={t('nav.close_menu')}
                onClick={() => setNavigationOpen(false)}
              >
                <svg
                  aria-hidden="true"
                  viewBox="0 0 20 20"
                  className="h-5 w-5"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.75"
                >
                  <path d="m5 5 10 10M15 5 5 15" strokeLinecap="round" />
                </svg>
              </button>
            </div>
            <nav className="space-y-1 p-3 text-sm text-zinc-600">{renderNavigationItems(true)}</nav>
            <div className="mt-auto border-t border-zinc-200 p-3">
              <LanguageSwitcher />
            </div>
          </aside>
        </>
      ) : null}
      <div className="flex min-h-screen min-w-0 flex-1 flex-col">
        <header className="flex min-h-14 items-center gap-2 border-b border-zinc-200 bg-white px-3 py-2 sm:px-4 lg:justify-end">
          <button
            type="button"
            className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md text-zinc-700 hover:bg-zinc-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent lg:hidden"
            aria-label={t('nav.open_menu')}
            aria-controls="primary-navigation"
            aria-expanded={navigationOpen}
            onClick={() => setNavigationOpen(true)}
          >
            <svg aria-hidden="true" viewBox="0 0 20 20" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="1.75">
              <path d="M3 5h14M3 10h14M3 15h14" strokeLinecap="round" />
            </svg>
          </button>
          <span className="font-semibold lg:hidden">{t('app.title')}</span>
          {user ? (
            <div className="ml-auto flex min-w-0 items-center gap-1 text-sm text-zinc-700 sm:gap-2 lg:ml-0 lg:gap-3">
              <Link
                to="/account"
                className="max-w-28 truncate font-medium text-zinc-900 hover:text-accent sm:max-w-48"
              >
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
          <div className="hidden lg:block">
            <LanguageSwitcher />
          </div>
        </header>
        <main className="min-w-0 flex-1 px-4 py-5 sm:p-6">
          <div className="mx-auto w-full min-w-0 max-w-7xl">{children}</div>
        </main>
        <AppVersion />
      </div>
    </div>
  );
}
