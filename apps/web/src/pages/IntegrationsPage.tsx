import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';
import {
  IntegrationConfigFields,
  buildConfigPayload,
  configFormFromItem,
  emptyIntegrationConfigForm,
  integrationFormReady,
  type IntegrationConfigForm,
  type IntegrationKind,
} from '../components/integrations/IntegrationConfigFields';
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
import type { Workspace } from '../lib/teamTypes';
import { fetchWorkspaces } from '../lib/workspacesApi';

type IntegrationItem = {
  id: string;
  kind: string;
  name: string;
  enabled: boolean;
  workspace_id?: string | null;
  config?: Record<string, unknown>;
  config_complete?: boolean;
};

type EditorMode = 'create' | 'edit';

type EditorState = {
  mode: EditorMode;
  id?: string;
  kind: IntegrationKind;
  name: string;
  workspace_id: string;
  form: IntegrationConfigForm;
};

const emptyEditor = (): EditorState => ({
  mode: 'create',
  kind: 'jira',
  name: '',
  workspace_id: '',
  form: emptyIntegrationConfigForm(),
});

export function IntegrationsPage() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [items, setItems] = useState<IntegrationItem[]>([]);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);
  const [editor, setEditor] = useState<EditorState | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<IntegrationItem | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const workspaceById = useMemo(
    () => new Map(workspaces.map((workspace) => [workspace.id, workspace])),
    [workspaces],
  );

  const loadIntegrations = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const response = await fetch('/api/v1/integrations', { credentials: 'include' });
      if (response.status === 401) {
        setLoadError(t('integrations.sign_in_required'));
        setItems([]);
        return;
      }
      if (!response.ok) {
        throw new Error(t('integrations.load_error'));
      }
      const data = (await response.json()) as { items: IntegrationItem[] };
      setItems(data.items ?? []);
    } catch {
      setLoadError(t('integrations.load_error'));
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void loadIntegrations();
    void fetchWorkspaces()
      .then(setWorkspaces)
      .catch(() => setWorkspaces([]));
  }, [loadIntegrations]);

  const testConnection = async (id: string) => {
    setTestingId(id);
    setToast(null);
    try {
      const response = await fetch(`/api/v1/integrations/${id}/test`, {
        method: 'POST',
        credentials: 'include',
      });
      if (response.status === 401) {
        setToast({ message: t('integrations.sign_in_required'), variant: 'default' });
        return;
      }
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('integrations.test_failed'));
      }
      setToast({ message: t('integrations.test_success'), variant: 'success' });
    } catch (error) {
      const message =
        error instanceof Error && error.message && error.message !== 'network'
          ? error.message
          : t('integrations.test_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setTestingId(null);
    }
  };

  const openCreate = () => setEditor(emptyEditor());

  const openEdit = (item: IntegrationItem) => {
    const kind = (['jira', 'slack', 'express'].includes(item.kind) ? item.kind : 'jira') as IntegrationKind;
    setEditor({
      mode: 'edit',
      id: item.id,
      kind,
      name: item.name,
      workspace_id: item.workspace_id ?? '',
      form: configFormFromItem(kind, item.config),
    });
  };

  const saveEditor = async () => {
    if (!editor) {
      return;
    }
    setSaving(true);
    setToast(null);
    try {
      const workspaceOnly = editor.workspace_id !== '' && editor.mode === 'create';
      const editing = editor.mode === 'edit';
      const config = buildConfigPayload(editor.kind, editor.form, {
        workspaceOnly: workspaceOnly || (editing && editor.workspace_id !== ''),
        keepBlankSecrets: editing,
      });

      if (editor.mode === 'create') {
        const payload: Record<string, unknown> = {
          kind: editor.kind,
          name: editor.name.trim() || editor.kind,
          enabled: true,
          config,
        };
        if (editor.workspace_id) {
          payload.workspace_id = editor.workspace_id;
        }
        const response = await fetch('/api/v1/integrations', {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
        if (response.status === 401) {
          setToast({ message: t('integrations.sign_in_required'), variant: 'default' });
          return;
        }
        if (!response.ok) {
          const body = (await response.json()) as { message?: string };
          throw new Error(body.message ?? t('integrations.save_failed'));
        }
      } else {
        const response = await fetch(`/api/v1/integrations/${editor.id}`, {
          method: 'PATCH',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: editor.name.trim() || editor.kind,
            config,
          }),
        });
        if (response.status === 401) {
          setToast({ message: t('integrations.sign_in_required'), variant: 'default' });
          return;
        }
        if (!response.ok) {
          const body = (await response.json()) as { message?: string };
          throw new Error(body.message ?? t('integrations.save_failed'));
        }
      }
      setToast({ message: t('integrations.save_success'), variant: 'success' });
      setEditor(null);
      await loadIntegrations();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('integrations.save_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setSaving(false);
    }
  };

  const toggleEnabled = async (item: IntegrationItem) => {
    setTogglingId(item.id);
    setToast(null);
    try {
      const response = await fetch(`/api/v1/integrations/${item.id}`, {
        method: 'PATCH',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: !item.enabled }),
      });
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('integrations.save_failed'));
      }
      setToast({
        message: item.enabled ? t('integrations.disabled_toast') : t('integrations.enabled_toast'),
        variant: 'success',
      });
      await loadIntegrations();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('integrations.save_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setTogglingId(null);
    }
  };

  const confirmDelete = async () => {
    if (!deleteTarget) {
      return;
    }
    setDeleting(true);
    setToast(null);
    try {
      const response = await fetch(`/api/v1/integrations/${deleteTarget.id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (!response.ok && response.status !== 204) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('integrations.delete_failed'));
      }
      setToast({ message: t('integrations.delete_success'), variant: 'success' });
      setDeleteTarget(null);
      await loadIntegrations();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('integrations.delete_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setDeleting(false);
    }
  };

  const renderScope = (item: IntegrationItem) => {
    if (!item.workspace_id) {
      return <StatusTag variant="neutral" label={t('integrations.scope.global')} />;
    }
    const workspace = workspaceById.get(item.workspace_id);
    return (
      <StatusTag
        variant="neutral"
        label={t('integrations.scope.workspace', { name: workspace?.name ?? item.workspace_id })}
      />
    );
  };

  const editorReady =
    editor !== null &&
    integrationFormReady(editor.kind, editor.form, {
      workspaceOnly: editor.workspace_id !== '',
      editing: editor.mode === 'edit',
    });

  const incompleteCount = items.filter((item) => item.config_complete === false).length;

  return (
    <PageContent>
      <PageHeader
        title={t('integrations.page_title')}
        subtitle={t('integrations.page_subtitle')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [
            { label: t('nav.platform'), href: '/dashboard' },
            { label: t('nav.integrations') },
          ],
        }}
        actions={isAdmin ? <Button onClick={openCreate}>{t('integrations.add')}</Button> : undefined}
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}
      {!loadError && incompleteCount > 0 ? (
        <Banner variant="warning">{t('integrations.incomplete_banner')}</Banner>
      ) : null}

      {loading ? (
        <p className="text-sm text-zinc-600">{t('integrations.loading')}</p>
      ) : loadError ? null : items.length === 0 ? (
        <p className="text-sm text-zinc-600">{t('integrations.empty')}</p>
      ) : (
        <DataTable
          columns={[
            {
              key: 'name',
              header: t('integrations.column.name'),
              cellClassName: 'font-medium text-zinc-900',
              render: (item) => (
                <div className="flex flex-col gap-1">
                  <span>{item.name}</span>
                  {item.config_complete === false ? (
                    <span className="text-xs font-normal text-severity-p2">{t('integrations.incomplete_status')}</span>
                  ) : null}
                </div>
              ),
            },
            {
              key: 'kind',
              header: t('integrations.column.kind'),
              cellClassName: 'text-zinc-700',
              render: (item) => item.kind,
            },
            {
              key: 'scope',
              header: t('integrations.column.scope'),
              render: (item) => renderScope(item),
            },
            {
              key: 'status',
              header: t('integrations.column.status'),
              cellClassName: 'text-zinc-700',
              render: (item) => (item.enabled ? t('integrations.enabled') : t('integrations.disabled')),
            },
            {
              key: 'actions',
              header: t('integrations.column.actions'),
              render: (item) => (
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="secondary"
                    disabled={testingId === item.id}
                    onClick={() => void testConnection(item.id)}
                  >
                    {testingId === item.id ? t('integrations.testing') : t('integrations.test_connection')}
                  </Button>
                  {isAdmin ? (
                    <>
                      <Button variant="secondary" onClick={() => openEdit(item)}>
                        {t('integrations.configure')}
                      </Button>
                      <Button
                        variant="secondary"
                        disabled={togglingId === item.id}
                        onClick={() => void toggleEnabled(item)}
                      >
                        {item.enabled ? t('integrations.disable') : t('integrations.enable')}
                      </Button>
                      <Button variant="ghost" onClick={() => setDeleteTarget(item)}>
                        {t('integrations.delete')}
                      </Button>
                    </>
                  ) : null}
                </div>
              ),
            },
          ]}
          rows={items}
          rowKey={(item) => item.id}
          emptyMessage={t('integrations.empty')}
        />
      )}

      {editor ? (
        <Modal
          title={editor.mode === 'create' ? t('integrations.add') : t('integrations.configure')}
          open
          onClose={() => setEditor(null)}
          primaryLabel={t('integrations.save')}
          secondaryLabel={t('teams.cancel')}
          onPrimary={() => void saveEditor()}
          primaryDisabled={saving || !editorReady}
          primaryLoading={saving}
        >
          <Select
            id="integration-kind"
            label={t('integrations.column.kind')}
            value={editor.kind}
            disabled={editor.mode === 'edit'}
            options={[
              { value: 'jira', label: 'Jira' },
              { value: 'slack', label: 'Slack' },
              { value: 'express', label: 'eXpress' },
            ]}
            onChange={(value) =>
              setEditor((current) =>
                current
                  ? {
                      ...current,
                      kind: value as IntegrationKind,
                      form: emptyIntegrationConfigForm(),
                    }
                  : current,
              )
            }
          />
          <Input
            label={t('integrations.column.name')}
            value={editor.name}
            onChange={(value) => setEditor((current) => (current ? { ...current, name: value } : current))}
          />
          {editor.mode === 'create' ? (
            <Select
              id="integration-workspace"
              label={t('integrations.workspace_label')}
              value={editor.workspace_id}
              options={[
                { value: '', label: t('integrations.scope.global') },
                ...workspaces.map((workspace) => ({ value: workspace.id, label: workspace.name })),
              ]}
              onChange={(value) =>
                setEditor((current) =>
                  current
                    ? {
                        ...current,
                        workspace_id: value,
                        kind: value ? 'jira' : current.kind,
                        form: emptyIntegrationConfigForm(),
                      }
                    : current,
                )
              }
            />
          ) : null}
          <IntegrationConfigFields
            kind={editor.kind}
            form={editor.form}
            workspaceOnly={editor.workspace_id !== ''}
            editing={editor.mode === 'edit'}
            onChange={(form) => setEditor((current) => (current ? { ...current, form } : current))}
          />
        </Modal>
      ) : null}

      <Modal
        title={t('integrations.delete_confirm_title')}
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        primaryLabel={t('integrations.delete')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void confirmDelete()}
        primaryDisabled={deleting}
        primaryLoading={deleting}
      >
        <p className="text-sm text-zinc-700">
          {t('integrations.delete_confirm_body', { name: deleteTarget?.name ?? '' })}
        </p>
      </Modal>

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
