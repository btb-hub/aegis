import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';
import { Banner } from '../components/ui/Banner';
import { Button } from '../components/ui/Button';
import { DataTable } from '../components/ui/DataTable';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Toast } from '../components/ui/Toast';
import {
  createWorkspace,
  deleteWorkspace,
  fetchWorkspaces,
  type WorkspaceSummary,
  WorkspaceApiError,
} from '../lib/workspacesApi';
import { DEFAULT_WORKSPACE_ID } from '../lib/teamTypes';

type WorkspaceFormState = {
  name: string;
  slug: string;
  description: string;
};

const emptyForm: WorkspaceFormState = {
  name: '',
  slug: '',
  description: '',
};

export function WorkspacesPage() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [items, setItems] = useState<WorkspaceSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);

  const [formOpen, setFormOpen] = useState(false);
  const [form, setForm] = useState<WorkspaceFormState>(emptyForm);
  const [saving, setSaving] = useState(false);

  const [deleteTarget, setDeleteTarget] = useState<WorkspaceSummary | null>(null);
  const [deleting, setDeleting] = useState(false);

  const loadItems = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      setItems(await fetchWorkspaces());
    } catch {
      setLoadError(t('workspaces.list.load_error'));
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void loadItems();
  }, [loadItems]);

  const openCreate = () => {
    setForm(emptyForm);
    setFormOpen(true);
  };

  const closeForm = () => {
    setFormOpen(false);
    setForm(emptyForm);
  };

  const saveWorkspace = async () => {
    setSaving(true);
    setToast(null);
    try {
      await createWorkspace({
        name: form.name.trim(),
        description: form.description.trim(),
        ...(form.slug.trim() ? { slug: form.slug.trim() } : {}),
      });
      setToast({ message: t('workspaces.list.create_success'), variant: 'success' });
      closeForm();
      await loadItems();
    } catch (error) {
      const message =
        error instanceof WorkspaceApiError ? error.message : t('workspaces.list.save_failed');
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
      await deleteWorkspace(deleteTarget.id);
      setToast({ message: t('workspaces.list.delete_success'), variant: 'success' });
      setDeleteTarget(null);
      await loadItems();
    } catch (error) {
      const message =
        error instanceof WorkspaceApiError ? error.message : t('workspaces.list.delete_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setDeleting(false);
    }
  };

  return (
    <PageContent>
      <PageHeader
        title={t('workspaces.list.page_title')}
        subtitle={t('workspaces.list.page_subtitle')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('nav.workspaces') }],
        }}
        actions={isAdmin ? <Button onClick={openCreate}>{t('workspaces.list.create')}</Button> : undefined}
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}

      {loading ? (
        <p className="text-sm text-zinc-600">{t('workspaces.loading')}</p>
      ) : loadError ? null : (
        <DataTable
          columns={[
            {
              key: 'name',
              header: t('workspaces.list.column.name'),
              cellClassName: 'font-medium text-zinc-900',
              render: (workspace) => (
                <Link to={`/workspaces/${workspace.id}`} className="text-accent hover:underline">
                  {workspace.name}
                </Link>
              ),
            },
            {
              key: 'slug',
              header: t('workspaces.list.column.slug'),
              cellClassName: 'text-zinc-700',
              render: (workspace) => workspace.slug,
            },
            {
              key: 'teams',
              header: t('workspaces.list.column.teams'),
              cellClassName: 'text-zinc-700',
              render: (workspace) => workspace.team_count,
            },
            {
              key: 'rules',
              header: t('workspaces.list.column.rules'),
              cellClassName: 'text-zinc-700',
              render: (workspace) => workspace.routing_rule_count,
            },
            {
              key: 'description',
              header: t('workspaces.list.column.description'),
              cellClassName: 'text-zinc-700 max-w-xs truncate',
              render: (workspace) => workspace.description || '—',
            },
            ...(isAdmin
              ? [
                  {
                    key: 'actions',
                    header: t('workspaces.list.column.actions'),
                    render: (workspace: WorkspaceSummary) =>
                      workspace.id === DEFAULT_WORKSPACE_ID ? (
                        '—'
                      ) : (
                        <Button variant="ghost" onClick={() => setDeleteTarget(workspace)}>
                          {t('workspaces.list.delete')}
                        </Button>
                      ),
                  },
                ]
              : []),
          ]}
          rows={items}
          rowKey={(workspace) => workspace.id}
          emptyMessage={t('workspaces.list.empty')}
        />
      )}

      <Modal
        title={t('workspaces.list.create')}
        open={formOpen}
        onClose={closeForm}
        primaryLabel={t('workspaces.list.save')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void saveWorkspace()}
        primaryDisabled={!form.name.trim() || saving}
        primaryLoading={saving}
      >
        <Input
          label={t('workspaces.list.name_label')}
          value={form.name}
          onChange={(value) => setForm((f) => ({ ...f, name: value }))}
        />
        <Input
          label={t('workspaces.list.slug_label')}
          value={form.slug}
          onChange={(value) => setForm((f) => ({ ...f, slug: value }))}
        />
        <Input
          label={t('workspaces.list.description_label')}
          value={form.description}
          onChange={(value) => setForm((f) => ({ ...f, description: value }))}
        />
      </Modal>

      <Modal
        title={t('workspaces.list.delete_confirm_title')}
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        primaryLabel={t('workspaces.list.delete')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void confirmDelete()}
        primaryDisabled={deleting}
        primaryLoading={deleting}
      >
        <p className="text-sm text-zinc-700">
          {t('workspaces.list.delete_confirm_body', { name: deleteTarget?.name ?? '' })}
        </p>
      </Modal>

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
