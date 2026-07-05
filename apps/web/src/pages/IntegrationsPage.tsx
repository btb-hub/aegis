import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';
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
};

type AddIntegrationForm = {
  kind: string;
  name: string;
  workspace_id: string;
  project_key: string;
};

const emptyAddForm: AddIntegrationForm = {
  kind: 'jira',
  name: '',
  workspace_id: '',
  project_key: '',
};

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
  const [addOpen, setAddOpen] = useState(false);
  const [addForm, setAddForm] = useState<AddIntegrationForm>(emptyAddForm);
  const [saving, setSaving] = useState(false);

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

  const saveIntegration = async () => {
    setSaving(true);
    setToast(null);
    try {
      const payload: Record<string, unknown> = {
        kind: addForm.kind,
        name: addForm.name.trim() || addForm.kind,
        enabled: true,
        config: addForm.workspace_id ? { project_key: addForm.project_key.trim() } : {},
      };
      if (addForm.workspace_id) {
        payload.workspace_id = addForm.workspace_id;
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
      setToast({ message: t('integrations.save_success'), variant: 'success' });
      setAddOpen(false);
      setAddForm(emptyAddForm);
      await loadIntegrations();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('integrations.save_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setSaving(false);
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
        actions={isAdmin ? <Button onClick={() => setAddOpen(true)}>{t('integrations.add')}</Button> : undefined}
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}

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
              render: (item) => item.name,
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
                <Button
                  variant="secondary"
                  disabled={testingId === item.id}
                  onClick={() => void testConnection(item.id)}
                >
                  {testingId === item.id ? t('integrations.testing') : t('integrations.test_connection')}
                </Button>
              ),
            },
          ]}
          rows={items}
          rowKey={(item) => item.id}
          emptyMessage={t('integrations.empty')}
        />
      )}

      <Modal
        title={t('integrations.add')}
        open={addOpen}
        onClose={() => setAddOpen(false)}
        primaryLabel={t('integrations.save')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void saveIntegration()}
        primaryDisabled={
          saving ||
          (addForm.workspace_id !== '' && !addForm.project_key.trim())
        }
        primaryLoading={saving}
      >
        <Select
          id="integration-kind"
          label={t('integrations.column.kind')}
          value={addForm.kind}
          options={[
            { value: 'jira', label: 'Jira' },
            { value: 'slack', label: 'Slack' },
            { value: 'express', label: 'eXpress' },
          ]}
          onChange={(value) => setAddForm((f) => ({ ...f, kind: value }))}
        />
        <Input
          label={t('integrations.column.name')}
          value={addForm.name}
          onChange={(value) => setAddForm((f) => ({ ...f, name: value }))}
        />
        <Select
          id="integration-workspace"
          label={t('integrations.workspace_label')}
          value={addForm.workspace_id}
          options={[
            { value: '', label: t('integrations.scope.global') },
            ...workspaces.map((workspace) => ({ value: workspace.id, label: workspace.name })),
          ]}
          onChange={(value) => setAddForm((f) => ({ ...f, workspace_id: value }))}
        />
        {addForm.workspace_id ? (
          <Input
            label={t('setup.integrations.jira.project_key')}
            value={addForm.project_key}
            onChange={(value) => setAddForm((f) => ({ ...f, project_key: value }))}
          />
        ) : null}
      </Modal>

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
