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
  | 'settings'
  | 'dashboard'
  | 'users';

type AppShellProps = {
  children: ReactNode;
  currentPage?: AppPage;
  onNavigate?: (page: AppPage) => void;
  user?: AuthUser | null;
  onSignOut?: () => void | Promise<void>;
};

const navigationIcons: Record<AppPage, ReactNode> = {
  dashboard: <><rect x="3" y="3" width="7" height="7" rx="1" /><rect x="14" y="3" width="7" height="7" rx="1" /><rect x="3" y="14" width="7" height="7" rx="1" /><rect x="14" y="14" width="7" height="7" rx="1" /></>,
  alerts: <><path d="M12 3 22 20H2L12 3Z" /><path d="M12 10v4" /><circle cx="12" cy="17" r=".7" fill="currentColor" /></>,
  incidents: <><circle cx="12" cy="12" r="9" /><path d="M12 7.5v5" /><circle cx="12" cy="16" r=".7" fill="currentColor" /></>,
  shifts: <><circle cx="12" cy="12" r="9" /><path d="M12 7v5l4 2" /></>,
  settings: <><path d="M4 7h16M4 12h16M4 17h16" /><circle cx="9" cy="7" r="2" fill="white" /><circle cx="15" cy="12" r="2" fill="white" /><circle cx="8" cy="17" r="2" fill="white" /></>,
  teams: <><circle cx="9" cy="8" r="3" /><path d="M3 20v-2a6 6 0 0 1 12 0v2M17 5a3 3 0 0 1 0 6m1 3a5 5 0 0 1 3 5v1" /></>,
  workspaces: <><rect x="3" y="5" width="18" height="15" rx="2" /><path d="M8 5V3h8v2M3 11h18" /></>,
  users: <><circle cx="12" cy="8" r="4" /><path d="M4 21a8 8 0 0 1 16 0" /></>,
};

function Brand() {
  return <span className="inline-flex items-center gap-2.5 font-bold text-zinc-900">
    <span className="flex h-7 w-7 items-center justify-center rounded-md bg-accent text-white">
      <svg aria-hidden="true" viewBox="0 0 24 24" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 3 19 5.5V11c0 4.4-3 8-7 10-4-2-7-5.6-7-10V5.5L12 3Z" />
      </svg>
    </span>
    Aegis
  </span>;
}

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
    ...(isAdmin ? [{ id: 'settings' as AppPage, label: t('nav.settings') }] : []),
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
      const classes = `flex w-full items-center gap-2.5 rounded-[7px] px-2.5 py-2 text-left text-[13px] ${
        active ? 'bg-blue-50 font-semibold text-blue-700' : 'font-medium text-zinc-500 hover:bg-white hover:text-zinc-900'
      }`;
      const content = <><svg aria-hidden="true" viewBox="0 0 24 24" className="h-[17px] w-[17px] shrink-0" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">{navigationIcons[item.id]}</svg><span>{item.label}</span></>;
      if (onNavigate) {
        return (
          <button
            key={item.id}
            type="button"
            className={classes}
            aria-current={active ? 'page' : undefined}
            onClick={() => {
              if (closeAfterNavigation) {
                setNavigationOpen(false);
              }
              onNavigate(item.id);
            }}
          >
            {content}
          </button>
        );
      }

      return (
        <div
          key={item.id}
          className={classes}
        >
          {content}
        </div>
      );
    });

  return (
    <div className="min-h-screen bg-zinc-50 lg:flex">
      <aside className="hidden w-60 shrink-0 flex-col border-r border-zinc-200 bg-zinc-50 lg:flex">
        <div className="flex h-14 items-center border-b border-zinc-200 px-4">
          <Brand />
        </div>
        <nav className="space-y-0.5 p-3">{renderNavigationItems(false)}</nav>
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
            className="fixed inset-y-0 left-0 z-40 flex w-60 flex-col border-r border-zinc-200 bg-zinc-50 shadow-xl lg:hidden"
          >
            <div className="flex h-14 items-center justify-between border-b border-zinc-200 px-4 font-semibold">
              <Brand />
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
            <nav className="space-y-0.5 p-3">{renderNavigationItems(true)}</nav>
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
          <span className="lg:hidden"><Brand /></span>
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
