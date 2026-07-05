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
import { Toast } from '../components/ui/Toast';
import { useAuth } from '../context/AuthContext';
import type { Team, TeamMember, TeamRole, UserDirectoryItem } from '../lib/teamTypes';
import { TEAM_ROLES } from '../lib/teamTypes';

const roleOptions = (t: (key: string) => string) =>
  TEAM_ROLES.map((role) => ({ value: role, label: t(`teams.detail.role.${role}`) }));

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
  const roles = useMemo(() => roleOptions(t), [t]);

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
          <Link to={`/teams/${teamId}/shifts`} className="text-sm font-medium text-accent hover:underline">
            {t('nav.shifts')}
          </Link>
        }
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}

      {loading ? (
        <p className="text-sm text-zinc-600">{t('teams.loading')}</p>
      ) : loadError ? null : (
        <>
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
        </>
      )}

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
