import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, Navigate } from 'react-router-dom';
import { Button } from '../components/ui/Button';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { fetchTeams } from '../lib/shiftsApi';
import type { Team } from '../lib/teamTypes';

export function ShiftsLandingPage() {
  const { t } = useTranslation();
  const [teams, setTeams] = useState<Team[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const items = await fetchTeams();
      setTeams(items);
    } catch {
      setError(t('shifts.load_error'));
      setTeams([]);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  if (teams === null) {
    return <p className="text-sm text-zinc-600">{t('shifts.loading')}</p>;
  }

  if (error) {
    return (
      <PageContent>
        <p className="text-sm text-red-700">{error}</p>
        <Button variant="secondary" onClick={() => void load()}>
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
