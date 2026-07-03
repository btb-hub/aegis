import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useParams } from 'react-router-dom';
import { TeamMemberPicker } from '../components/teams/TeamMemberPicker';
import { Button } from '../components/ui/Button';
import { PageBreadcrumb } from '../components/ui/PageBreadcrumb';
import { Toast } from '../components/ui/Toast';
import { useAuth } from '../context/AuthContext';
import type { Team, TeamMember, TeamRole, UserDirectoryItem } from '../lib/teamTypes';
import { TEAM_ROLES } from '../lib/teamTypes';

export function TeamDetailPage() {
  const { t } = useTranslation();
  const { teamId = '' } = useParams();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [team, setTeam] = useState<Team | null>(null);
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);
  const [pendingRole, setPendingRole] = useState<TeamRole>('member');
  const [addingMember, setAddingMember] = useState(false);
  const [updatingUserId, setUpdatingUserId] = useState<string | null>(null);
  const [removingUserId, setRemovingUserId] = useState<string | null>(null);

  const memberUserIds = useMemo(() => members.map((member) => member.user_id), [members]);

  const loadTeam = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const [teamResponse, membersResponse] = await Promise.all([
        fetch(`/api/v1/teams/${teamId}`, { credentials: 'include' }),
        fetch(`/api/v1/teams/${teamId}/members`, { credentials: 'include' }),
      ]);
      if (teamResponse.status === 401 || membersResponse.status === 401) {
        setLoadError(t('teams.sign_in_required'));
        setTeam(null);
        setMembers([]);
        return;
      }
      if (!teamResponse.ok || !membersResponse.ok) {
        throw new Error(t('teams.detail.load_error'));
      }
      setTeam((await teamResponse.json()) as Team);
      const membersData = (await membersResponse.json()) as { items: TeamMember[] };
      setMembers(membersData.items ?? []);
    } catch {
      setLoadError(t('teams.detail.load_error'));
      setTeam(null);
      setMembers([]);
    } finally {
      setLoading(false);
    }
  }, [teamId, t]);

  useEffect(() => {
    void loadTeam();
  }, [loadTeam]);

  const addMember = async (selected: UserDirectoryItem) => {
    setAddingMember(true);
    setToast(null);
    try {
      const response = await fetch(`/api/v1/teams/${teamId}/members`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: selected.id, team_role: pendingRole }),
      });
      if (response.status === 401) {
        setToast({ message: t('teams.sign_in_required'), variant: 'default' });
        return;
      }
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('teams.detail.member_add_failed'));
      }
      setToast({ message: t('teams.detail.member_add_success'), variant: 'success' });
      await loadTeam();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('teams.detail.member_add_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setAddingMember(false);
    }
  };

  const updateMemberRole = async (member: TeamMember, teamRole: TeamRole) => {
    setUpdatingUserId(member.user_id);
    setToast(null);
    try {
      const response = await fetch(`/api/v1/teams/${teamId}/members/${member.user_id}`, {
        method: 'PATCH',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ team_role: teamRole }),
      });
      if (response.status === 401) {
        setToast({ message: t('teams.sign_in_required'), variant: 'default' });
        return;
      }
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('teams.detail.member_update_failed'));
      }
      setToast({ message: t('teams.detail.member_update_success'), variant: 'success' });
      await loadTeam();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('teams.detail.member_update_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setUpdatingUserId(null);
    }
  };

  const removeMember = async (member: TeamMember) => {
    setRemovingUserId(member.user_id);
    setToast(null);
    try {
      const response = await fetch(`/api/v1/teams/${teamId}/members/${member.user_id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (response.status === 401) {
        setToast({ message: t('teams.sign_in_required'), variant: 'default' });
        return;
      }
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('teams.detail.member_remove_failed'));
      }
      setToast({ message: t('teams.detail.member_remove_success'), variant: 'success' });
      await loadTeam();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('teams.detail.member_remove_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setRemovingUserId(null);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <PageBreadcrumb
          ariaLabel={t('nav.breadcrumb_label')}
          items={[
            { label: t('nav.teams'), href: '/teams' },
            { label: team?.name ?? t('teams.detail.loading_name') },
          ]}
        />
        <h1 className="text-2xl font-semibold text-zinc-900">{team?.name ?? t('teams.detail.loading_name')}</h1>
        {team?.description ? <p className="mt-1 text-sm text-zinc-600">{team.description}</p> : null}
      </div>

      {loadError ? (
        <div
          role="alert"
          className="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950"
        >
          {loadError}
        </div>
      ) : null}

      {loading ? (
        <p className="text-sm text-zinc-600">{t('teams.loading')}</p>
      ) : loadError ? null : (
        <>
          <section className="space-y-4">
            <div className="flex items-center justify-between gap-4">
              <h2 className="text-lg font-semibold text-zinc-900">{t('teams.detail.members_title')}</h2>
              <Link to={`/teams/${teamId}/shifts`} className="text-sm text-accent hover:underline">
                {t('nav.shifts')}
              </Link>
            </div>

            {members.length === 0 ? (
              <p className="text-sm text-zinc-600">{t('teams.detail.members_empty')}</p>
            ) : (
              <div className="overflow-hidden rounded-lg border border-zinc-200 bg-white">
                <table className="min-w-full divide-y divide-zinc-200 text-sm">
                  <thead className="bg-zinc-50 text-left text-zinc-600">
                    <tr>
                      <th className="px-4 py-3 font-medium">{t('teams.detail.column.name')}</th>
                      <th className="px-4 py-3 font-medium">{t('teams.detail.column.email')}</th>
                      <th className="px-4 py-3 font-medium">{t('teams.detail.column.role')}</th>
                      {isAdmin ? (
                        <th className="px-4 py-3 font-medium">{t('teams.detail.column.actions')}</th>
                      ) : null}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-200">
                    {members.map((member) => (
                      <tr key={member.id}>
                        <td className="px-4 py-3 font-medium text-zinc-900">{member.display_name}</td>
                        <td className="px-4 py-3 text-zinc-700">{member.email}</td>
                        <td className="px-4 py-3 text-zinc-700">
                          {isAdmin ? (
                            <select
                              aria-label={t('teams.detail.role_label')}
                              className="h-9 rounded-md border border-zinc-300 px-2 text-sm"
                              value={member.team_role}
                              disabled={updatingUserId === member.user_id}
                              onChange={(event) =>
                                void updateMemberRole(member, event.target.value as TeamRole)
                              }
                            >
                              {TEAM_ROLES.map((role) => (
                                <option key={role} value={role}>
                                  {t(`teams.detail.role.${role}`)}
                                </option>
                              ))}
                            </select>
                          ) : (
                            t(`teams.detail.role.${member.team_role}`)
                          )}
                        </td>
                        {isAdmin ? (
                          <td className="px-4 py-3">
                            <Button
                              variant="ghost"
                              disabled={removingUserId === member.user_id}
                              onClick={() => void removeMember(member)}
                            >
                              {t('teams.detail.remove_member')}
                            </Button>
                          </td>
                        ) : null}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          {isAdmin ? (
            <section className="space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-4">
              <h3 className="text-sm font-semibold text-zinc-900">{t('teams.detail.add_member')}</h3>
              <label className="block text-sm text-zinc-700">
                <span className="mb-1 block font-medium">{t('teams.detail.role_label')}</span>
                <select
                  className="h-9 rounded-md border border-zinc-300 px-2 text-sm"
                  value={pendingRole}
                  onChange={(event) => setPendingRole(event.target.value as TeamRole)}
                >
                  {TEAM_ROLES.map((role) => (
                    <option key={role} value={role}>
                      {t(`teams.detail.role.${role}`)}
                    </option>
                  ))}
                </select>
              </label>
              <TeamMemberPicker
                disabled={addingMember}
                excludeUserIds={memberUserIds}
                onSelect={(selected) => void addMember(selected)}
              />
            </section>
          ) : null}
        </>
      )}

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </div>
  );
}
