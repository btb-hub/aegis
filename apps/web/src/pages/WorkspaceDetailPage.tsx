import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useParams } from 'react-router-dom';
import { WorkspaceSlotsPanel } from '../components/integrations/WorkspaceSlotsPanel';
import { Banner } from '../components/ui/Banner';
import { Button } from '../components/ui/Button';
import { Checkbox } from '../components/ui/Checkbox';
import { DataTable } from '../components/ui/DataTable';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Select } from '../components/ui/Select';
import { Toast } from '../components/ui/Toast';
import { useAuth } from '../context/AuthContext';
import { fetchTeams } from '../lib/shiftsApi';
import type { Team, Workspace } from '../lib/teamTypes';
import {
  assignTeamsToWorkspace,
  createRoutingRule,
  deleteRoutingRule,
  fetchRoutingRules,
  fetchWorkspace,
  updateRoutingRule,
  updateWorkspace,
  WorkspaceApiError,
  type RoutingRule,
} from '../lib/workspacesApi';

type RuleFormState = {
  team_id: string;
  priority: string;
  labelKey: string;
  labelValue: string;
};

type WorkspaceFormState = {
  name: string;
  slug: string;
  description: string;
};

const emptyRuleForm: RuleFormState = {
  team_id: '',
  priority: '100',
  labelKey: '',
  labelValue: '',
};

export function WorkspaceDetailPage() {
  const { t } = useTranslation();
  const { workspaceId = '' } = useParams();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [teams, setTeams] = useState<Team[]>([]);
  const [allTeams, setAllTeams] = useState<Team[]>([]);
  const [rules, setRules] = useState<RoutingRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);

  const [formOpen, setFormOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<RoutingRule | null>(null);
  const [form, setForm] = useState<RuleFormState>(emptyRuleForm);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<RoutingRule | null>(null);
  const [deleting, setDeleting] = useState(false);

  const [editOpen, setEditOpen] = useState(false);
  const [editForm, setEditForm] = useState<WorkspaceFormState>({ name: '', slug: '', description: '' });
  const [editSaving, setEditSaving] = useState(false);

  const [assignOpen, setAssignOpen] = useState(false);
  const [assignSearch, setAssignSearch] = useState('');
  const [selectedTeamIds, setSelectedTeamIds] = useState<string[]>([]);
  const [assignSaving, setAssignSaving] = useState(false);
  const [assignError, setAssignError] = useState<string | null>(null);

  const teamById = useMemo(() => new Map(teams.map((team) => [team.id, team])), [teams]);
  const teamOptions = useMemo(
    () => teams.map((team) => ({ value: team.id, label: team.name })),
    [teams],
  );

  const workspaceRules = useMemo(
    () => rules.filter((rule) => teamById.has(rule.team_id)),
    [rules, teamById],
  );

  const assignableTeams = useMemo(() => {
    const query = assignSearch.trim().toLowerCase();
    return allTeams.filter((team) => {
      if (team.workspace_id === workspaceId) {
        return false;
      }
      if (!query) {
        return true;
      }
      return team.name.toLowerCase().includes(query);
    });
  }, [allTeams, assignSearch, workspaceId]);

  const loadData = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const [workspaceData, teamList, ruleList] = await Promise.all([
        fetchWorkspace(workspaceId),
        fetchTeams(),
        fetchRoutingRules(),
      ]);
      setWorkspace(workspaceData);
      setAllTeams(teamList);
      setTeams(teamList.filter((team) => team.workspace_id === workspaceId));
      setRules(ruleList);
    } catch {
      setLoadError(t('workspaces.detail.load_error'));
      setWorkspace(null);
      setTeams([]);
      setAllTeams([]);
      setRules([]);
    } finally {
      setLoading(false);
    }
  }, [workspaceId, t]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const openCreate = () => {
    setEditingRule(null);
    setForm({
      ...emptyRuleForm,
      team_id: teams[0]?.id ?? '',
    });
    setFormOpen(true);
  };

  const openEditWorkspace = () => {
    if (!workspace) {
      return;
    }
    setEditForm({
      name: workspace.name,
      slug: workspace.slug,
      description: workspace.description ?? '',
    });
    setEditOpen(true);
  };

  const openAssign = () => {
    setAssignSearch('');
    setSelectedTeamIds([]);
    setAssignError(null);
    setAssignOpen(true);
  };

  const toggleAssignTeam = (teamId: string, checked: boolean) => {
    setSelectedTeamIds((current) =>
      checked ? [...current, teamId] : current.filter((id) => id !== teamId),
    );
  };

  const saveWorkspaceMeta = async () => {
    if (!workspace) {
      return;
    }
    setEditSaving(true);
    setToast(null);
    try {
      const updated = await updateWorkspace(workspace.id, {
        name: editForm.name.trim(),
        slug: editForm.slug.trim(),
        description: editForm.description.trim(),
      });
      setWorkspace(updated);
      setToast({ message: t('workspaces.detail.update_success'), variant: 'success' });
      setEditOpen(false);
    } catch (error) {
      const message =
        error instanceof WorkspaceApiError ? error.message : t('workspaces.detail.update_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setEditSaving(false);
    }
  };

  const saveAssignedTeams = async () => {
    setAssignSaving(true);
    setAssignError(null);
    try {
      await assignTeamsToWorkspace(workspaceId, selectedTeamIds);
      setToast({ message: t('workspaces.teams.assign_success'), variant: 'success' });
      setAssignOpen(false);
      await loadData();
    } catch (error) {
      if (error instanceof WorkspaceApiError && error.status === 409) {
        setAssignError(t('workspaces.teams.assign_blocked'));
      } else {
        setAssignError(
          error instanceof WorkspaceApiError ? error.message : t('workspaces.teams.assign_failed'),
        );
      }
    } finally {
      setAssignSaving(false);
    }
  };

  const openEdit = (rule: RoutingRule) => {
    const entries = Object.entries(rule.match_labels ?? {});
    const [labelKey = '', labelValue = ''] = entries[0] ?? [];
    setEditingRule(rule);
    setForm({
      team_id: rule.team_id,
      priority: String(rule.priority),
      labelKey,
      labelValue,
    });
    setFormOpen(true);
  };

  const closeForm = () => {
    setFormOpen(false);
    setEditingRule(null);
    setForm(emptyRuleForm);
  };

  const saveRule = async () => {
    setSaving(true);
    setToast(null);
    try {
      const priority = Number.parseInt(form.priority, 10);
      const payload = {
        team_id: form.team_id,
        priority: Number.isFinite(priority) ? priority : 100,
        match_labels: { [form.labelKey.trim()]: form.labelValue.trim() },
      };
      if (editingRule) {
        await updateRoutingRule(editingRule.id, payload);
      } else {
        await createRoutingRule(payload);
      }
      setToast({
        message: t(editingRule ? 'workspaces.routing.update_success' : 'workspaces.routing.create_success'),
        variant: 'success',
      });
      closeForm();
      setRules(await fetchRoutingRules());
    } catch (error) {
      const message = error instanceof Error ? error.message : t('workspaces.routing.save_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setSaving(false);
    }
  };

  const confirmDelete = async () => {
    if (!deleteTarget) {
      return;
    }
    setDeleting(true);
    setToast(null);
    try {
      await deleteRoutingRule(deleteTarget.id);
      setToast({ message: t('workspaces.routing.delete_success'), variant: 'success' });
      setDeleteTarget(null);
      setRules(await fetchRoutingRules());
    } catch (error) {
      const message = error instanceof Error ? error.message : t('workspaces.routing.delete_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setDeleting(false);
    }
  };

  const formatMatchers = (rule: RoutingRule) =>
    Object.entries(rule.match_labels ?? {})
      .map(([key, value]) => `${key}=${value}`)
      .join(', ') || '—';

  const workspaceName = workspace?.name ?? t('workspaces.detail.loading_name');

  return (
    <PageContent>
      <PageHeader
        title={workspaceName}
        subtitle={workspace?.description ?? undefined}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [
            { label: t('nav.platform'), href: '/dashboard' },
            { label: t('nav.workspaces'), href: '/workspaces' },
            { label: workspaceName },
          ],
        }}
        actions={
          isAdmin ? (
            <div className="flex flex-wrap gap-2">
              <Button variant="secondary" onClick={openEditWorkspace}>
                {t('workspaces.detail.edit')}
              </Button>
              <Button onClick={openCreate}>{t('workspaces.routing.create')}</Button>
            </div>
          ) : undefined
        }
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}

      {loading ? (
        <p className="text-sm text-zinc-600">{t('workspaces.loading')}</p>
      ) : loadError ? null : (
        <div className="space-y-10">
          <section className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h2 className="text-lg font-semibold text-zinc-900">{t('workspaces.teams.title')}</h2>
                <p className="text-sm text-zinc-600">{t('workspaces.teams.subtitle')}</p>
              </div>
              {isAdmin ? (
                <Button variant="secondary" onClick={openAssign}>
                  {t('workspaces.teams.add_existing')}
                </Button>
              ) : null}
            </div>
            <DataTable
              columns={[
                {
                  key: 'name',
                  header: t('teams.column.name'),
                  cellClassName: 'font-medium text-zinc-900',
                  render: (team) => (
                    <Link to={`/teams/${team.id}`} className="text-accent hover:underline">
                      {team.name}
                    </Link>
                  ),
                },
                {
                  key: 'tier',
                  header: t('teams.column.tier'),
                  cellClassName: 'text-zinc-700',
                  render: (team) =>
                    team.support_tier ? t(`teams.tier.${team.support_tier}`) : '—',
                },
                {
                  key: 'description',
                  header: t('teams.column.description'),
                  cellClassName: 'text-zinc-700',
                  render: (team) => team.description || '—',
                },
              ]}
              rows={teams}
              rowKey={(team) => team.id}
              emptyMessage={t('workspaces.teams.empty')}
            />
          </section>

          <WorkspaceSlotsPanel workspaceId={workspaceId} isAdmin={isAdmin} />

          <section className="space-y-4">
            <h2 className="text-lg font-semibold text-zinc-900">{t('workspaces.routing.title')}</h2>
            <p className="text-sm text-zinc-600">{t('workspaces.routing.subtitle')}</p>
            <DataTable
              columns={[
                {
                  key: 'team',
                  header: t('workspaces.routing.column.team'),
                  cellClassName: 'font-medium text-zinc-900',
                  render: (rule) => {
                    const team = teamById.get(rule.team_id);
                    // workspaceRules only includes rows whose team_id is in teamById
                    return (
                      <Link to={`/teams/${team!.id}`} className="text-accent hover:underline">
                        {team!.name}
                      </Link>
                    );
                  },
                },
                {
                  key: 'matchers',
                  header: t('workspaces.routing.column.matchers'),
                  cellClassName: 'text-zinc-700',
                  render: (rule) => formatMatchers(rule),
                },
                {
                  key: 'priority',
                  header: t('workspaces.routing.column.priority'),
                  cellClassName: 'text-zinc-700',
                  render: (rule) => rule.priority,
                },
                ...(isAdmin
                  ? [
                      {
                        key: 'actions',
                        header: t('workspaces.routing.column.actions'),
                        render: (rule: RoutingRule) => (
                          <div className="flex flex-wrap gap-2">
                            <Button variant="ghost" onClick={() => openEdit(rule)}>
                              {t('workspaces.routing.edit')}
                            </Button>
                            <Button variant="ghost" onClick={() => setDeleteTarget(rule)}>
                              {t('workspaces.routing.delete')}
                            </Button>
                          </div>
                        ),
                      },
                    ]
                  : []),
              ]}
              rows={workspaceRules}
              rowKey={(rule) => rule.id}
              emptyMessage={t('workspaces.routing.empty')}
            />
          </section>
        </div>
      )}

      <Modal
        title={t('workspaces.detail.edit')}
        open={editOpen}
        onClose={() => setEditOpen(false)}
        primaryLabel={t('workspaces.list.save')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void saveWorkspaceMeta()}
        primaryDisabled={!editForm.name.trim() || editSaving}
        primaryLoading={editSaving}
      >
        <Input
          label={t('workspaces.list.name_label')}
          value={editForm.name}
          onChange={(value) => setEditForm((f) => ({ ...f, name: value }))}
        />
        <Input
          label={t('workspaces.list.slug_label')}
          value={editForm.slug}
          onChange={(value) => setEditForm((f) => ({ ...f, slug: value }))}
        />
        <Input
          label={t('workspaces.list.description_label')}
          value={editForm.description}
          onChange={(value) => setEditForm((f) => ({ ...f, description: value }))}
        />
      </Modal>

      <Modal
        title={t('workspaces.teams.add_existing')}
        open={assignOpen}
        onClose={() => setAssignOpen(false)}
        primaryLabel={t('workspaces.teams.move')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void saveAssignedTeams()}
        primaryDisabled={selectedTeamIds.length === 0 || assignSaving}
        primaryLoading={assignSaving}
      >
        <Input
          label={t('workspaces.teams.search_label')}
          value={assignSearch}
          onChange={setAssignSearch}
        />
        {assignError ? <Banner variant="warning">{assignError}</Banner> : null}
        <div className="max-h-64 space-y-2 overflow-y-auto">
          {assignableTeams.length === 0 ? (
            <p className="text-sm text-zinc-600">{t('workspaces.teams.no_candidates')}</p>
          ) : (
            assignableTeams.map((team) => (
              <Checkbox
                key={team.id}
                id={`assign-team-${team.id}`}
                label={team.name}
                checked={selectedTeamIds.includes(team.id)}
                onChange={(checked) => toggleAssignTeam(team.id, checked)}
              />
            ))
          )}
        </div>
      </Modal>

      <Modal
        title={t(editingRule ? 'workspaces.routing.edit' : 'workspaces.routing.create')}
        open={formOpen}
        onClose={closeForm}
        primaryLabel={t('workspaces.routing.save')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void saveRule()}
        primaryDisabled={
          !form.team_id || !form.labelKey.trim() || !form.labelValue.trim() || saving
        }
        primaryLoading={saving}
      >
        <Select
          id="routing-team"
          label={t('workspaces.routing.team_label')}
          value={form.team_id}
          options={teamOptions}
          onChange={(value) => setForm((f) => ({ ...f, team_id: value }))}
        />
        <Input
          label={t('workspaces.routing.label_key')}
          value={form.labelKey}
          onChange={(value) => setForm((f) => ({ ...f, labelKey: value }))}
        />
        <Input
          label={t('workspaces.routing.label_value')}
          value={form.labelValue}
          onChange={(value) => setForm((f) => ({ ...f, labelValue: value }))}
        />
        <Input
          label={t('workspaces.routing.priority_label')}
          value={form.priority}
          onChange={(value) => setForm((f) => ({ ...f, priority: value }))}
        />
      </Modal>

      <Modal
        title={t('workspaces.routing.delete_confirm_title')}
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        primaryLabel={t('workspaces.routing.delete')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void confirmDelete()}
        primaryDisabled={deleting}
        primaryLoading={deleting}
      >
        <p className="text-sm text-zinc-700">{t('workspaces.routing.delete_confirm_body')}</p>
      </Modal>

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
