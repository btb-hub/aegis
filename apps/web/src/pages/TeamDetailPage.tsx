import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useParams } from 'react-router-dom';
import { TeamMemberPicker } from '../components/teams/TeamMemberPicker';
import { Banner } from '../components/ui/Banner';
import { Button } from '../components/ui/Button';
import { DataTable } from '../components/ui/DataTable';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Select } from '../components/ui/Select';
import { StatusTag } from '../components/ui/StatusTag';
import { Toast } from '../components/ui/Toast';
import { useAuth } from '../context/AuthContext';
import { resolveApiErrorMessage } from '../lib/apiErrors';
import { fetchTeams } from '../lib/shiftsApi';
import type { Team, TeamMember, TeamRole, UserDirectoryItem } from '../lib/teamTypes';
import { SUPPORT_TIERS, TEAM_ROLES, validEscalationTargetTiers } from '../lib/teamTypes';
import {
  addEscalationPath,
  deleteEscalationPath,
  fetchIncomingPaths,
  fetchOutgoingPaths,
  type EscalationPath,
} from '../lib/workspacesApi';

const roleOptions = (t: (key: string) => string) =>
  TEAM_ROLES.map((role) => ({ value: role, label: t(`teams.detail.role.${role}`) }));

const tierOptions = (t: (key: string) => string) => [
  { value: '', label: t('teams.tier.unset') },
  ...SUPPORT_TIERS.map((tier) => ({ value: tier, label: t(`teams.tier.${tier}`) })),
];

export function TeamDetailPage() {
  const { t } = useTranslation();
  const { teamId = '' } = useParams();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [team, setTeam] = useState<Team | null>(null);
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [workspaceTeams, setWorkspaceTeams] = useState<Team[]>([]);
  const [outgoingPaths, setOutgoingPaths] = useState<EscalationPath[]>([]);
  const [incomingPaths, setIncomingPaths] = useState<EscalationPath[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);
  const [pendingRole, setPendingRole] = useState<TeamRole>('member');
  const [addingMember, setAddingMember] = useState(false);
  const [updatingUserId, setUpdatingUserId] = useState<string | null>(null);
  const [removingUserId, setRemovingUserId] = useState<string | null>(null);
  const [pendingTier, setPendingTier] = useState('');
  const [savingTier, setSavingTier] = useState(false);
  const [targetTeamId, setTargetTeamId] = useState('');
  const [addingPath, setAddingPath] = useState(false);
  const [removingPathId, setRemovingPathId] = useState<string | null>(null);

  const memberUserIds = useMemo(() => members.map((member) => member.user_id), [members]);
  const roles = useMemo(() => roleOptions(t), [t]);
  const tiers = useMemo(() => tierOptions(t), [t]);

  const teamById = useMemo(
    () => new Map(workspaceTeams.map((item) => [item.id, item])),
    [workspaceTeams],
  );

  const validTargetTeams = useMemo(() => {
    if (!team?.support_tier) {
      return [];
    }
    const allowedTiers = validEscalationTargetTiers(team.support_tier);
    return workspaceTeams.filter(
      (item) =>
        item.id !== team.id &&
        item.support_tier &&
        allowedTiers.includes(item.support_tier) &&
        !outgoingPaths.some((path) => path.to_team_id === item.id),
    );
  }, [team, workspaceTeams, outgoingPaths]);

  const targetTeamOptions = useMemo(
    () => validTargetTeams.map((item) => ({ value: item.id, label: item.name })),
    [validTargetTeams],
  );

  const loadTeam = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const [teamResponse, membersResponse, outgoing, incoming] = await Promise.all([
        fetch(`/api/v1/teams/${teamId}`, { credentials: 'include' }),
        fetch(`/api/v1/teams/${teamId}/members`, { credentials: 'include' }),
        fetchOutgoingPaths(teamId),
        fetchIncomingPaths(teamId),
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
      const teamData = (await teamResponse.json()) as Team;
      const membersData = (await membersResponse.json()) as { items: TeamMember[] };
      const allTeams = await fetchTeams();
      setTeam(teamData);
      setMembers(membersData.items ?? []);
      setWorkspaceTeams(allTeams.filter((item) => item.workspace_id === teamData.workspace_id));
      setOutgoingPaths(outgoing);
      setIncomingPaths(incoming);
      setPendingTier(teamData.support_tier ?? '');
      setTargetTeamId('');
    } catch {
      setLoadError(t('teams.detail.load_error'));
      setTeam(null);
      setMembers([]);
      setWorkspaceTeams([]);
      setOutgoingPaths([]);
      setIncomingPaths([]);
    } finally {
      setLoading(false);
    }
  }, [teamId, t]);

  useEffect(() => {
    void loadTeam();
  }, [loadTeam]);

  useEffect(() => {
    if (validTargetTeams.length > 0 && !targetTeamId) {
      setTargetTeamId(validTargetTeams[0].id);
    }
  }, [validTargetTeams, targetTeamId]);

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
        throw new Error(
          resolveApiErrorMessage(t, body, t('teams.detail.member_add_failed')),
        );
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
        throw new Error(
          resolveApiErrorMessage(t, body, t('teams.detail.member_update_failed')),
        );
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
        throw new Error(
          resolveApiErrorMessage(t, body, t('teams.detail.member_remove_failed')),
        );
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

  const saveSupportTier = async () => {
    setSavingTier(true);
    setToast(null);
    try {
      const response = await fetch(`/api/v1/teams/${teamId}`, {
        method: 'PATCH',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ support_tier: pendingTier }),
      });
      if (response.status === 401) {
        setToast({ message: t('teams.sign_in_required'), variant: 'default' });
        return;
      }
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(resolveApiErrorMessage(t, body, t('teams.detail.tier_update_failed')));
      }
      setToast({ message: t('teams.detail.tier_update_success'), variant: 'success' });
      await loadTeam();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('teams.detail.tier_update_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setSavingTier(false);
    }
  };

  const createPath = async () => {
    if (!team || !targetTeamId) {
      return;
    }
    setAddingPath(true);
    setToast(null);
    try {
      await addEscalationPath(team.workspace_id, {
        from_team_id: team.id,
        to_team_id: targetTeamId,
      });
      setToast({ message: t('teams.detail.path_add_success'), variant: 'success' });
      await loadTeam();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('teams.detail.path_add_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setAddingPath(false);
    }
  };

  const removePath = async (pathId: string) => {
    setRemovingPathId(pathId);
    setToast(null);
    try {
      await deleteEscalationPath(pathId);
      setToast({ message: t('teams.detail.path_remove_success'), variant: 'success' });
      await loadTeam();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('teams.detail.path_remove_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setRemovingPathId(null);
    }
  };

  const renderPathTeam = (id: string) => {
    const pathTeam = teamById.get(id);
    if (!pathTeam) {
      return id;
    }
    return (
      <Link to={`/teams/${pathTeam.id}`} className="text-accent hover:underline">
        {pathTeam.name}
      </Link>
    );
  };

  const teamName = team?.name ?? t('teams.detail.loading_name');

  return (
    <PageContent>
      <PageHeader
        title={teamName}
        subtitle={team?.description ?? undefined}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [
            { label: t('nav.platform'), href: '/dashboard' },
            { label: t('nav.teams'), href: '/teams' },
            { label: teamName },
          ],
        }}
        actions={
          <div className="flex flex-wrap items-center gap-3">
            {team?.workspace_id ? (
              <Link
                to={`/workspaces/${team.workspace_id}`}
                className="text-sm font-medium text-accent hover:underline"
              >
                {t('teams.detail.open_workspace')}
              </Link>
            ) : null}
            <Link to={`/teams/${teamId}/shifts`} className="text-sm font-medium text-accent hover:underline">
              {t('nav.shifts')}
            </Link>
          </div>
        }
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}

      {loading ? (
        <p className="text-sm text-zinc-600">{t('teams.loading')}</p>
      ) : loadError ? null : (
        <>
          <section className="space-y-3 rounded-lg border border-zinc-200 bg-white p-4">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-lg font-semibold text-zinc-900">{t('teams.detail.settings_title')}</h2>
              {team?.support_tier ? (
                <StatusTag variant="neutral" label={t(`teams.tier.${team.support_tier}`)} />
              ) : null}
            </div>
            {isAdmin ? (
              <div className="flex flex-wrap items-end gap-3">
                <div className="min-w-[12rem]">
                  <Select
                    id="team-support-tier"
                    label={t('teams.tier_label')}
                    value={pendingTier}
                    options={tiers}
                    disabled={savingTier}
                    onChange={setPendingTier}
                  />
                </div>
                <Button disabled={savingTier} onClick={() => void saveSupportTier()}>
                  {t('teams.detail.save_tier')}
                </Button>
              </div>
            ) : team?.support_tier ? (
              <p className="text-sm text-zinc-700">{t(`teams.tier.${team.support_tier}`)}</p>
            ) : (
              <p className="text-sm text-zinc-600">{t('teams.tier.unset')}</p>
            )}
          </section>

          <section className="space-y-4">
            <h2 className="text-lg font-semibold text-zinc-900">{t('teams.detail.members_title')}</h2>
            <DataTable
              columns={[
                {
                  key: 'name',
                  header: t('teams.detail.column.name'),
                  cellClassName: 'font-medium text-zinc-900',
                  render: (member) => member.display_name,
                },
                {
                  key: 'email',
                  header: t('teams.detail.column.email'),
                  cellClassName: 'text-zinc-700',
                  render: (member) => member.email,
                },
                {
                  key: 'role',
                  header: t('teams.detail.column.role'),
                  render: (member) =>
                    isAdmin ? (
                      <Select
                        hideLabel
                        id={`member-role-${member.user_id}`}
                        label={t('teams.detail.role_label')}
                        value={member.team_role}
                        disabled={updatingUserId === member.user_id}
                        options={roles}
                        onChange={(value) => void updateMemberRole(member, value as TeamRole)}
                      />
                    ) : (
                      t(`teams.detail.role.${member.team_role}`)
                    ),
                },
                ...(isAdmin
                  ? [
                      {
                        key: 'actions',
                        header: t('teams.detail.column.actions'),
                        render: (member: TeamMember) => (
                          <Button
                            variant="ghost"
                            disabled={removingUserId === member.user_id}
                            onClick={() => void removeMember(member)}
                          >
                            {t('teams.detail.remove_member')}
                          </Button>
                        ),
                      },
                    ]
                  : []),
              ]}
              rows={members}
              rowKey={(member) => member.id}
              emptyMessage={t('teams.detail.members_empty')}
            />
          </section>

          {isAdmin ? (
            <section className="space-y-3 rounded-lg border border-zinc-200 bg-white p-4">
              <h3 className="text-sm font-semibold text-zinc-900">{t('teams.detail.add_member')}</h3>
              <Select
                id="add-member-role"
                label={t('teams.detail.role_label')}
                value={pendingRole}
                options={roles}
                onChange={(value) => setPendingRole(value as TeamRole)}
              />
              <TeamMemberPicker
                disabled={addingMember}
                excludeUserIds={memberUserIds}
                onSelect={(selected) => void addMember(selected)}
              />
            </section>
          ) : null}

          <section className="space-y-4">
            <h2 className="text-lg font-semibold text-zinc-900">{t('teams.detail.escalation_title')}</h2>
            <p className="text-sm text-zinc-600">{t('teams.detail.escalation_subtitle')}</p>

            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-zinc-900">{t('teams.detail.outgoing_paths')}</h3>
              <DataTable
                columns={[
                  {
                    key: 'target',
                    header: t('teams.detail.path_target'),
                    render: (path) => renderPathTeam(path.to_team_id),
                  },
                  ...(isAdmin
                    ? [
                        {
                          key: 'actions',
                          header: t('teams.detail.column.actions'),
                          render: (path: EscalationPath) => (
                            <Button
                              variant="ghost"
                              disabled={removingPathId === path.id}
                              onClick={() => void removePath(path.id)}
                            >
                              {t('teams.detail.remove_path')}
                            </Button>
                          ),
                        },
                      ]
                    : []),
                ]}
                rows={outgoingPaths}
                rowKey={(path) => path.id}
                emptyMessage={t('teams.detail.paths_empty')}
              />
            </div>

            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-zinc-900">{t('teams.detail.incoming_paths')}</h3>
              <DataTable
                columns={[
                  {
                    key: 'source',
                    header: t('teams.detail.path_source'),
                    render: (path) => renderPathTeam(path.from_team_id),
                  },
                ]}
                rows={incomingPaths}
                rowKey={(path) => path.id}
                emptyMessage={t('teams.detail.paths_empty')}
              />
            </div>

            {isAdmin && team?.support_tier ? (
              <div className="space-y-3 rounded-lg border border-zinc-200 bg-zinc-50 p-4">
                <h3 className="text-sm font-semibold text-zinc-900">{t('teams.detail.add_path')}</h3>
                {validTargetTeams.length === 0 ? (
                  <p className="text-sm text-zinc-600">{t('teams.detail.no_valid_targets')}</p>
                ) : (
                  <>
                    <Select
                      id="escalation-target-team"
                      label={t('teams.detail.path_target')}
                      value={targetTeamId}
                      options={targetTeamOptions}
                      onChange={setTargetTeamId}
                    />
                    <Button disabled={addingPath || !targetTeamId} onClick={() => void createPath()}>
                      {t('teams.detail.add_path')}
                    </Button>
                  </>
                )}
              </div>
            ) : null}
          </section>
        </>
      )}

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
