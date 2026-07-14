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
  type IntegrationMode,
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
  mode?: IntegrationMode;
  slot_status?: 'ready' | 'needs_setup' | 'using_global' | 'missing' | 'disabled';
};

type EditorMode = 'create' | 'edit';

type EditorState = {
  mode: EditorMode;
  id?: string;
  kind: IntegrationKind;
  name: string;
  workspace_id: string;
  form: IntegrationConfigForm;
  integrationMode?: IntegrationMode;
  savedIntegrationMode?: IntegrationMode;
};

const integrationKinds: IntegrationKind[] = ['jira', 'slack', 'express'];
const kindNames: Record<IntegrationKind, string> = {
  jira: 'Jira',
  slack: 'Slack',
  express: 'eXpress',
};

const emptyEditor = (kind: IntegrationKind): EditorState => ({
  mode: 'create',
  kind,
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
  const [scopeFilter, setScopeFilter] = useState('all');
  const [kindFilter, setKindFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');

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

  const missingGlobalKinds = useMemo(() => {
    const configured = new Set(items.filter((item) => !item.workspace_id).map((item) => item.kind));
    return integrationKinds.filter((kind) => !configured.has(kind));
  }, [items]);

  const filteredItems = useMemo(
    () =>
      items.filter((item) => {
        const scope = item.workspace_id ? 'workspace' : 'global';
        const status = item.workspace_id ? item.slot_status : item.enabled ? 'enabled' : 'disabled';
        return (
          (scopeFilter === 'all' || scopeFilter === scope) &&
          (kindFilter === 'all' || kindFilter === item.kind) &&
          (statusFilter === 'all' || statusFilter === status)
        );
      }),
    [items, kindFilter, scopeFilter, statusFilter],
  );

  const openCreate = () => {
    const kind = missingGlobalKinds[0];
    if (kind) {
      setEditor(emptyEditor(kind));
    }
  };

  const openEdit = (item: IntegrationItem) => {
    const kind = (['jira', 'slack', 'express'].includes(item.kind) ? item.kind : 'jira') as IntegrationKind;
    setEditor({
      mode: 'edit',
      id: item.id,
      kind,
      name: item.name,
      workspace_id: item.workspace_id ?? '',
      form: configFormFromItem(kind, item.config),
      integrationMode: item.workspace_id ? (item.mode ?? 'inherit') : undefined,
      savedIntegrationMode: item.workspace_id ? (item.mode ?? 'inherit') : undefined,
    });
  };

  const changeIntegrationMode = (mode: IntegrationMode) => {
    setEditor((current) => {
      if (!current || current.integrationMode === mode) {
        return current;
      }
      if (
        current.integrationMode === 'custom' &&
        mode === 'inherit' &&
        !window.confirm(t('workspaces.integrations.switch_confirm'))
      ) {
        return current;
      }
      const form =
        mode === 'custom'
          ? emptyIntegrationConfigForm()
          : {
              ...emptyIntegrationConfigForm(),
              project_key: current.form.project_key,
            };
      return { ...current, integrationMode: mode, form };
    });
  };

  const saveEditor = async () => {
    if (!editor) {
      return;
    }
    setSaving(true);
    setToast(null);
    try {
      const editing = editor.mode === 'edit';
      const workspaceMode = editor.integrationMode;
      const config =
        workspaceMode === 'inherit'
          ? editor.kind === 'jira'
            ? { project_key: editor.form.project_key.trim() }
            : {}
          : buildConfigPayload(editor.kind, editor.form, {
              workspaceOnly: false,
              keepBlankSecrets: editing && (workspaceMode === undefined || editor.savedIntegrationMode === 'custom'),
            });

      if (editor.mode === 'create') {
        const payload: Record<string, unknown> = {
          kind: editor.kind,
          name: editor.name.trim() || editor.kind,
          enabled: true,
          config,
        };
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
          body: JSON.stringify(
            editor.workspace_id
              ? {
                  mode: workspaceMode,
                  enabled: items.find((item) => item.id === editor.id)?.enabled ?? true,
                  config,
                }
              : {
                  name: editor.name.trim() || editor.kind,
                  config,
                },
          ),
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

  const renderStatus = (item: IntegrationItem) => {
    if (item.workspace_id && item.slot_status) {
      return (
        <StatusTag
          variant={item.slot_status === 'ready' || item.slot_status === 'using_global' ? 'resolved' : 'neutral'}
          label={t(`workspaces.integrations.status_${item.slot_status}`)}
        />
      );
    }
    return item.enabled ? t('integrations.enabled') : t('integrations.disabled');
  };

  const editorReady =
    editor !== null &&
    (editor.integrationMode === 'inherit'
      ? editor.kind !== 'jira' || editor.form.project_key.trim() !== ''
      : integrationFormReady(editor.kind, editor.form, {
          workspaceOnly: false,
          editing:
            editor.mode === 'edit' &&
            (editor.integrationMode === undefined || editor.savedIntegrationMode === 'custom'),
        }));

  const incompleteCount = items.filter(
    (item) => item.config_complete === false && (!item.workspace_id || item.mode === 'custom'),
  ).length;

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
        actions={
          isAdmin && missingGlobalKinds.length > 0 ? (
            <Button onClick={openCreate}>{t('integrations.add')}</Button>
          ) : undefined
        }
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}
      {!loadError && incompleteCount > 0 ? (
        <Banner variant="warning">{t('integrations.incomplete_banner')}</Banner>
      ) : null}

      {!loading && !loadError && items.length > 0 ? (
        <div className="grid gap-3 sm:grid-cols-3">
          <Select
            id="integration-scope-filter"
            label={t('integrations.filter.scope')}
            value={scopeFilter}
            options={[
              { value: 'all', label: t('integrations.filter.all_scopes') },
              { value: 'global', label: t('integrations.scope.global') },
              { value: 'workspace', label: t('integrations.filter.workspace') },
            ]}
            onChange={setScopeFilter}
          />
          <Select
            id="integration-kind-filter"
            label={t('integrations.filter.kind')}
            value={kindFilter}
            options={[
              { value: 'all', label: t('integrations.filter.all_kinds') },
              ...integrationKinds.map((kind) => ({ value: kind, label: kindNames[kind] })),
            ]}
            onChange={setKindFilter}
          />
          <Select
            id="integration-status-filter"
            label={t('integrations.filter.status')}
            value={statusFilter}
            options={[
              { value: 'all', label: t('integrations.filter.all_statuses') },
              { value: 'enabled', label: t('integrations.enabled') },
              { value: 'disabled', label: t('integrations.disabled') },
              { value: 'ready', label: t('workspaces.integrations.status_ready') },
              { value: 'needs_setup', label: t('workspaces.integrations.status_needs_setup') },
              { value: 'using_global', label: t('workspaces.integrations.status_using_global') },
              { value: 'missing', label: t('workspaces.integrations.status_missing') },
            ]}
            onChange={setStatusFilter}
          />
        </div>
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
              render: (item) => renderStatus(item),
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
                      {!item.workspace_id ? (
                        <Button variant="ghost" onClick={() => setDeleteTarget(item)}>
                          {t('integrations.delete')}
                        </Button>
                      ) : null}
                    </>
                  ) : null}
                </div>
              ),
            },
          ]}
          rows={filteredItems}
          rowKey={(item) => item.id}
          emptyMessage={t('integrations.empty_filtered')}
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
            options={(editor.mode === 'create' ? missingGlobalKinds : integrationKinds).map((kind) => ({
              value: kind,
              label: kindNames[kind],
            }))}
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
          {!editor.workspace_id ? (
            <Input
              label={t('integrations.column.name')}
              value={editor.name}
              onChange={(value) => setEditor((current) => (current ? { ...current, name: value } : current))}
            />
          ) : null}
          {editor.workspace_id && editor.integrationMode ? (
            <>
              <Select
                id="workspace-integration-mode"
                label={t('workspaces.integrations.mode')}
                value={editor.integrationMode}
                options={[
                  { value: 'inherit', label: t('workspaces.integrations.mode_inherit') },
                  { value: 'custom', label: t('workspaces.integrations.mode_custom') },
                ]}
                onChange={(value) => changeIntegrationMode(value as IntegrationMode)}
              />
              <p className="text-sm text-zinc-600">
                {t(
                  editor.integrationMode === 'inherit'
                    ? 'workspaces.integrations.inherit_help'
                    : 'workspaces.integrations.custom_help',
                )}
              </p>
            </>
          ) : null}
          <IntegrationConfigFields
            kind={editor.kind}
            form={editor.form}
            mode={editor.integrationMode}
            workspaceOnly={editor.workspace_id !== ''}
            editing={
              editor.mode === 'edit' && (!editor.workspace_id || editor.savedIntegrationMode === 'custom')
            }
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
