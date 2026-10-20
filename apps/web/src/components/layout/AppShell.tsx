import { useEffect, useRef, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { LanguageSwitcher } from './LanguageSwitcher';
import { AppVersion } from './AppVersion';
import type { AuthUser } from '../../lib/authTypes';

export type AppPage =
  | 'anomaly'
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
  anomaly: <><path d="M2 12h4l3-8 5 16 3-8h5" /></>,
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
  return <span className="inline-flex items-center gap-3 text-lg font-semibold tracking-tight text-navigation-text">
    <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent text-white">
      <svg aria-hidden="true" viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 3 19 5.5V11c0 4.4-3 8-7 10-4-2-7-5.6-7-10V5.5L12 3Z" />
      </svg>
    </span>
    Aegis
  </span>;
}

export function AppShell({ children, currentPage = 'shifts', onNavigate, user, onSignOut }: AppShellProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const accountActive = useLocation().pathname.startsWith('/account');
  const [navigationOpen, setNavigationOpen] = useState(false);
  const menuButton = useRef<HTMLButtonElement>(null);

  const groups: Array<{ label: string; pages: AppPage[] }> = [
    { label: t('nav.operations'), pages: ['incidents', 'alerts', 'shifts', 'anomaly', 'dashboard'] },
    { label: t('nav.organization'), pages: ['teams', 'workspaces', ...(user?.role === 'admin' ? ['users' as AppPage] : [])] },
    ...(user?.role === 'admin' ? [{ label: t('nav.administration'), pages: ['settings' as AppPage] }] : []),
  ];

  useEffect(() => { setNavigationOpen(false); }, [currentPage]);

  useEffect(() => {
    if (!navigationOpen) {
      return;
    }
    const close = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setNavigationOpen(false);
        menuButton.current?.focus();
      }
    };
    window.addEventListener('keydown', close);
    return () => window.removeEventListener('keydown', close);
  }, [navigationOpen]);

  const navigation = (mobile: boolean) => (
    <nav aria-label={t('nav.primary')} className="space-y-6 px-3 py-5">
      {groups.map((group) => (
        <div key={group.label}>
          <p className="mb-2 px-3 text-xs font-medium tracking-wide text-navigation-muted">{group.label}</p>
          <div className="space-y-1">
            {group.pages.map((page) => {
              const active = !accountActive && currentPage === page;
              const classes = `flex min-h-11 items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors focus-visible:outline-white ${active ? 'bg-accent font-semibold text-white' : 'text-navigation-muted hover:bg-white/10 hover:text-white'}`;
              const content = <><svg aria-hidden="true" viewBox="0 0 24 24" className="h-[18px] w-[18px] shrink-0" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">{navigationIcons[page]}</svg><span>{t(`nav.${page}`)}</span></>;
              if (page === 'anomaly') {
                return <a key={page} href="/modules/anomaly-detector/" className={classes}
                  aria-current={active ? 'page' : undefined} onClick={() => setNavigationOpen(false)}>
                  {content}
                </a>;
              }
              return <Link key={page} to={`/${page}`} className={classes} aria-current={active ? 'page' : undefined} onClick={(event) => {
                if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
                  return;
                }
                setNavigationOpen(false);
                if (onNavigate) {
                  event.preventDefault();
                  onNavigate(page);
                }
                if (mobile) {
                  document.getElementById('main-content')?.focus();
                }
              }}>{content}</Link>;
            })}
          </div>
        </div>
      ))}
    </nav>
  );

  const account = (
    <div className="mt-auto space-y-4 border-t border-white/15 p-4">
      {user ? <div className="flex min-w-0 items-center gap-3">
        <span aria-hidden="true" className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-white/10 text-sm font-semibold text-white">{(user.display_name || user.email).slice(0, 2).toUpperCase()}</span>
        <Link to="/account" aria-current={accountActive ? 'page' : undefined} onClick={(event) => {
          if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
            return;
          }
          if (navigationOpen) {
            document.getElementById('main-content')?.focus();
          }
          setNavigationOpen(false);
        }} className="min-w-0 flex-1 break-words text-sm font-medium text-navigation-text hover:underline focus-visible:outline-white">{user.display_name || user.email}</Link>
        <button type="button" className="min-h-11 shrink-0 rounded-md px-2 text-xs text-navigation-muted hover:bg-white/10 hover:text-white focus-visible:outline-white" onClick={() => { void (async () => { await onSignOut?.(); navigate('/login'); })(); }}>{t('auth.sign_out')}</button>
      </div> : null}
      <LanguageSwitcher appearance="dark" />
    </div>
  );

  return (
    <div className="min-h-screen bg-canvas lg:flex">
      <a href="#main-content" className="fixed left-4 top-3 z-50 -translate-y-24 rounded-md bg-accent px-4 py-3 font-medium text-white focus:translate-y-0">{t('nav.skip_to_content')}</a>
      <aside className="sticky top-0 hidden h-dvh w-60 shrink-0 flex-col overflow-y-auto bg-navigation lg:flex">
        <div className="px-6 pb-3 pt-7"><Brand /><p className="mt-2 text-xs text-navigation-muted">{t('app.tagline')}</p></div>
        {navigation(false)}
        {account}
      </aside>
      <div className="flex min-h-screen min-w-0 flex-1 flex-col">
        <header className="border-b border-white/15 bg-navigation text-navigation-text lg:hidden">
          <div className="flex min-h-16 items-center justify-between gap-3 px-4">
            <Brand />
            <button ref={menuButton} type="button" className="inline-flex h-11 w-11 items-center justify-center rounded-md hover:bg-white/10 focus-visible:outline-white" aria-label={navigationOpen ? t('nav.close_menu') : t('nav.open_menu')} aria-controls="primary-navigation" aria-expanded={navigationOpen} onClick={() => setNavigationOpen(!navigationOpen)}>
              <svg aria-hidden="true" viewBox="0 0 20 20" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="1.75"><path d={navigationOpen ? 'm5 5 10 10M15 5 5 15' : 'M3 5h14M3 10h14M3 15h14'} strokeLinecap="round" /></svg>
            </button>
          </div>
          {navigationOpen ? <div id="primary-navigation" className="border-t border-white/15">{navigation(true)}{account}</div> : null}
        </header>
        <main id="main-content" tabIndex={-1} className="min-w-0 flex-1 px-[16px] py-5 focus:outline-none sm:p-6 lg:px-8 lg:py-7">
          <div className="mx-auto w-full min-w-0 max-w-7xl">{children}</div>
        </main>
        <AppVersion />
      </div>
    </div>
  );
}
