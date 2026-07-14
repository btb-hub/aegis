import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import {
  DEFAULT_WORKSPACE_ID,
  SUPPORT_TIERS,
  type SupportTier,
  type Team,
  type Workspace,
} from '../lib/teamTypes';
import { fetchWorkspaces } from '../lib/workspacesApi';
import { Banner } from '../components/ui/Banner';
import { Button } from '../components/ui/Button';
import { DataTable } from '../components/ui/DataTable';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Select } from '../components/ui/Select';
import { StatusTag } from '../components/ui/StatusTag';
import { Toast } from '../components/ui/Toast';

type TeamFormState = {
  name: string;
  description: string;
  workspace_id: string;
  support_tier: string;
};

const emptyForm: TeamFormState = {
  name: '',
  description: '',
  workspace_id: DEFAULT_WORKSPACE_ID,
  support_tier: '',
};

export function TeamsPage() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [items, setItems] = useState<Team[]>([]);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [workspaceFilter, setWorkspaceFilter] = useState('all');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);

  const [formOpen, setFormOpen] = useState(false);
  const [editingTeam, setEditingTeam] = useState<Team | null>(null);
  const [form, setForm] = useState<TeamFormState>(emptyForm);
  const [saving, setSaving] = useState(false);

  const [deleteTarget, setDeleteTarget] = useState<Team | null>(null);
  const [deleting, setDeleting] = useState(false);

  const workspaceById = useMemo(
    () => new Map(workspaces.map((workspace) => [workspace.id, workspace])),
    [workspaces],
  );

  const tierOptions = useMemo(
    () => [
      { value: '', label: t('teams.tier.unset') },
      ...SUPPORT_TIERS.map((tier) => ({ value: tier, label: t(`teams.tier.${tier}`) })),
    ],
    [t],
  );

  const workspaceOptions = useMemo(
    () => workspaces.map((workspace) => ({ value: workspace.id, label: workspace.name })),
    [workspaces],
  );

  const loadWorkspaces = useCallback(async () => {
    try {
      const list = await fetchWorkspaces();
      setWorkspaces(list);
    } catch {
      setWorkspaces([]);
    }
  }, []);

  const loadTeams = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const query =
        workspaceFilter !== 'all'
          ? `/api/v1/teams?workspace_id=${encodeURIComponent(workspaceFilter)}`
          : '/api/v1/teams';
      const response = await fetch(query, { credentials: 'include' });
      if (response.status === 401) {
        setLoadError(t('teams.sign_in_required'));
        setItems([]);
        return;
      }
      if (!response.ok) {
        throw new Error(t('teams.load_error'));
      }
      const data = (await response.json()) as { items: Team[] };
      setItems(data.items ?? []);
    } catch {
      setLoadError(t('teams.load_error'));
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [t, workspaceFilter]);

  useEffect(() => {
    void loadWorkspaces();
  }, [loadWorkspaces]);

  useEffect(() => {
    void loadTeams();
  }, [loadTeams]);

  const openCreate = () => {
    setEditingTeam(null);
    setForm({
      ...emptyForm,
      workspace_id: workspaceFilter !== 'all' ? workspaceFilter : DEFAULT_WORKSPACE_ID,
    });
    setFormOpen(true);
  };

  const openEdit = (team: Team) => {
    setEditingTeam(team);
    setForm({
      name: team.name,
      description: team.description ?? '',
      workspace_id: team.workspace_id,
      support_tier: team.support_tier ?? '',
    });
    setFormOpen(true);
  };

  const closeForm = () => {
    setFormOpen(false);
    setEditingTeam(null);
    setForm(emptyForm);
  };

  const saveTeam = async () => {
    setSaving(true);
    setToast(null);
    try {
      const payload: Record<string, string> = {
        name: form.name.trim(),
        description: form.description.trim(),
      };
      if (editingTeam) {
        if (form.support_tier) {
          payload.support_tier = form.support_tier;
        } else {
          payload.support_tier = '';
        }
        payload.workspace_id = form.workspace_id;
      } else {
        payload.workspace_id = form.workspace_id;
        if (form.support_tier) {
          payload.support_tier = form.support_tier;
        }
      }

      const response = await fetch(
        editingTeam ? `/api/v1/teams/${editingTeam.id}` : '/api/v1/teams',
        {
          method: editingTeam ? 'PATCH' : 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        },
      );
      if (response.status === 401) {
        setToast({ message: t('teams.sign_in_required'), variant: 'default' });
        return;
      }
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('teams.save_failed'));
      }
      setToast({
        message: t(editingTeam ? 'teams.update_success' : 'teams.create_success'),
        variant: 'success',
      });
      closeForm();
      await loadTeams();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('teams.save_failed');
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
      const response = await fetch(`/api/v1/teams/${deleteTarget.id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (response.status === 401) {
        setToast({ message: t('teams.sign_in_required'), variant: 'default' });
        return;
      }
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('teams.delete_failed'));
      }
      setToast({ message: t('teams.delete_success'), variant: 'success' });
      setDeleteTarget(null);
      await loadTeams();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('teams.delete_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setDeleting(false);
    }
  };

  const renderTier = (tier?: SupportTier) =>
    tier ? (
      <StatusTag variant="neutral" label={t(`teams.tier.${tier}`)} />
    ) : (
      <span className="text-zinc-500">—</span>
    );

  return (
    <PageContent>
      <PageHeader
        title={t('teams.page_title')}
        subtitle={t('teams.page_subtitle')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('nav.teams') }],
        }}
        actions={isAdmin ? <Button onClick={openCreate}>{t('teams.create')}</Button> : undefined}
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}

      <div className="mb-4 max-w-xs">
        <Select
          id="teams-workspace-filter"
          label={t('teams.workspace_filter')}
          value={workspaceFilter}
          options={[
            { value: 'all', label: t('teams.workspace_all') },
            ...workspaces.map((workspace) => ({ value: workspace.id, label: workspace.name })),
          ]}
          onChange={setWorkspaceFilter}
        />
      </div>

      {loading ? (
        <p className="text-sm text-zinc-600">{t('teams.loading')}</p>
      ) : loadError ? null : items.length === 0 ? (
        <div className="rounded-lg border border-dashed border-zinc-300 bg-zinc-50 px-6 py-10 text-center">
          <p className="text-sm text-zinc-600">{t('teams.empty')}</p>
          {isAdmin ? (
            <div className="mt-4">
              <Button onClick={openCreate}>{t('teams.empty_cta')}</Button>
            </div>
          ) : null}
        </div>
      ) : (
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
              key: 'workspace',
              header: t('teams.column.workspace'),
              cellClassName: 'text-zinc-700',
              render: (team) => {
                const workspace = workspaceById.get(team.workspace_id);
                if (!workspace) {
                  return '—';
                }
                return (
                  <Link to={`/workspaces/${workspace.id}`} className="text-accent hover:underline">
                    {workspace.name}
                  </Link>
                );
              },
            },
            {
              key: 'tier',
              header: t('teams.column.tier'),
              render: (team) => renderTier(team.support_tier),
            },
            {
              key: 'description',
              header: t('teams.column.description'),
              cellClassName: 'text-zinc-700',
              render: (team) => team.description || '—',
            },
            {
              key: 'actions',
              header: t('teams.column.actions'),
              render: (team) =>
                isAdmin ? (
                  <div className="flex flex-wrap gap-2">
                    <Button variant="ghost" onClick={() => openEdit(team)}>
                      {t('teams.edit')}
                    </Button>
                    <Button variant="ghost" onClick={() => setDeleteTarget(team)}>
                      {t('teams.delete')}
                    </Button>
                  </div>
                ) : (
                  '—'
                ),
            },
          ]}
          rows={items}
          rowKey={(team) => team.id}
          emptyMessage={t('teams.empty')}
        />
      )}

      <Modal
        title={t(editingTeam ? 'teams.edit' : 'teams.create')}
        open={formOpen}
        onClose={closeForm}
        primaryLabel={t('teams.save')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void saveTeam()}
        primaryDisabled={!form.name.trim() || saving || !form.workspace_id}
        primaryLoading={saving}
      >
        <Select
          id="team-workspace"
          label={t('teams.workspace_label')}
          value={form.workspace_id}
          options={workspaceOptions}
          onChange={(value) => setForm((f) => ({ ...f, workspace_id: value }))}
        />
        <Select
          id="team-support-tier"
          label={t('teams.tier_label')}
          value={form.support_tier}
          options={tierOptions}
          onChange={(value) => setForm((f) => ({ ...f, support_tier: value }))}
        />
        <Input label={t('teams.name_label')} value={form.name} onChange={(value) => setForm((f) => ({ ...f, name: value }))} />
        <Input
          label={t('teams.description_label')}
          value={form.description}
          onChange={(value) => setForm((f) => ({ ...f, description: value }))}
        />
      </Modal>

      <Modal
        title={t('teams.delete_confirm_title')}
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        primaryLabel={t('teams.delete')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void confirmDelete()}
        primaryDisabled={deleting}
        primaryLoading={deleting}
      >
        <p className="text-sm text-zinc-700">{t('teams.delete_confirm_body')}</p>
      </Modal>

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
