import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useParams } from 'react-router-dom';
import { Banner } from '../components/ui/Banner';
import { Button } from '../components/ui/Button';
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
  createRoutingRule,
  deleteRoutingRule,
  fetchRoutingRules,
  fetchWorkspace,
  updateRoutingRule,
  type RoutingRule,
} from '../lib/workspacesApi';

type RuleFormState = {
  team_id: string;
  priority: string;
  labelKey: string;
  labelValue: string;
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

  const teamById = useMemo(() => new Map(teams.map((team) => [team.id, team])), [teams]);
  const teamOptions = useMemo(
    () => teams.map((team) => ({ value: team.id, label: team.name })),
    [teams],
  );

  const workspaceRules = useMemo(
    () => rules.filter((rule) => teamById.has(rule.team_id)),
    [rules, teamById],
  );

  const loadData = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const [workspaceData, teamList, ruleList] = await Promise.all([
        fetchWorkspace(workspaceId),
        fetchTeams().then((items) => items.filter((team) => team.workspace_id === workspaceId)),
        fetchRoutingRules(),
      ]);
      setWorkspace(workspaceData);
      setTeams(teamList);
      setRules(ruleList);
    } catch {
      setLoadError(t('workspaces.detail.load_error'));
      setWorkspace(null);
      setTeams([]);
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
            { label: t('nav.teams'), href: '/teams' },
            { label: workspaceName },
          ],
        }}
        actions={
          isAdmin ? (
            <Button onClick={openCreate}>{t('workspaces.routing.create')}</Button>
          ) : undefined
        }
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}

      {loading ? (
        <p className="text-sm text-zinc-600">{t('workspaces.loading')}</p>
      ) : loadError ? null : (
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
                  if (!team) {
                    return rule.team_id;
                  }
                  return (
                    <Link to={`/teams/${team.id}`} className="text-accent hover:underline">
                      {team.name}
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
      )}

      <Modal
        title={t(editingRule ? 'workspaces.routing.edit' : 'workspaces.routing.create')}
        open={formOpen}
        onClose={closeForm}
        primaryLabel={t('workspaces.routing.save')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void saveRule()}
        primaryDisabled={
          !form.team_id ||
          !form.labelKey.trim() ||
          !form.labelValue.trim() ||
          saving
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
