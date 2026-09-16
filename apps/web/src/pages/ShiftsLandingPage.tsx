import { useTranslation } from 'react-i18next';
import { Link, Navigate } from 'react-router-dom';
import { Button } from '../components/ui/Button';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { queryKeys } from '../lib/queryClient';
import { fetchTeams } from '../lib/shiftsApi';
import { useLoader } from '../lib/useLoader';

export function ShiftsLandingPage() {
  const { t } = useTranslation();
  const teamsQuery = useLoader(queryKeys.shifts.landing, () => fetchTeams());
  const teams = teamsQuery.data;
  const error = teamsQuery.isError ? t('shifts.load_error') : null;

  if (teamsQuery.loading) {
    return <p className="text-sm text-zinc-600">{t('shifts.loading')}</p>;
  }

  if (error || teams === undefined) {
    return (
      <PageContent>
        <p className="text-sm text-red-700">{error}</p>
        <Button variant="secondary" onClick={() => void teamsQuery.refetch()}>
          {t('shifts.retry')}
        </Button>
      </PageContent>
    );
  }

  if (teams.length === 1) {
    return <Navigate to={`/teams/${teams[0].id}/shifts`} replace />;
  }

  if (teams.length === 0) {
    return (
      <PageContent>
        <PageHeader
          title={t('nav.shifts')}
          subtitle={t('shifts.no_teams')}
          breadcrumb={{
            ariaLabel: t('nav.breadcrumb_label'),
            items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('nav.shifts') }],
          }}
        />
        <Link to="/teams" className="text-sm text-accent hover:underline">
          {t('shifts.no_teams_cta')}
        </Link>
      </PageContent>
    );
  }

  return (
    <PageContent>
      <PageHeader
        title={t('nav.shifts')}
        subtitle={t('shifts.select_team')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('nav.shifts') }],
        }}
      />
      <ul className="divide-y divide-zinc-200 rounded-lg border border-zinc-200 bg-white">
        {teams.map((team) => (
          <li key={team.id}>
            <Link
              to={`/teams/${team.id}/shifts`}
              className="block px-4 py-3 text-sm font-medium text-accent hover:bg-zinc-50"
            >
              {team.name}
            </Link>
          </li>
        ))}
      </ul>
    </PageContent>
  );
}
