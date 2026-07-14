import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  IntegrationConfigFields,
  buildConfigPayload,
  configFormFromItem,
  emptyIntegrationConfigForm,
  integrationFormReady,
  type IntegrationConfigForm,
  type IntegrationKind,
  type IntegrationMode,
} from './IntegrationConfigFields';
import { Banner } from '../ui/Banner';
import { Button } from '../ui/Button';
import { DataTable } from '../ui/DataTable';
import { Modal } from '../ui/Modal';
import { Select } from '../ui/Select';
import { StatusTag } from '../ui/StatusTag';
import { Toast } from '../ui/Toast';

type SlotStatus = 'ready' | 'needs_setup' | 'using_global' | 'missing' | 'disabled';

type IntegrationSlot = {
  id?: string;
  workspace_id: string;
  kind: IntegrationKind;
  name: string;
  enabled: boolean;
  mode: IntegrationMode;
  slot_status: SlotStatus;
  config: Record<string, unknown>;
};

type EditorState = {
  slot: IntegrationSlot;
  mode: IntegrationMode;
  form: IntegrationConfigForm;
};

type Props = {
  workspaceId: string;
  isAdmin: boolean;
};

const kinds: IntegrationKind[] = ['jira', 'slack', 'express'];
const kindNames: Record<IntegrationKind, string> = {
  jira: 'Jira',
  slack: 'Slack',
  express: 'eXpress',
};

function placeholderSlot(workspaceId: string, kind: IntegrationKind): IntegrationSlot {
  return {
    workspace_id: workspaceId,
    kind,
    name: kindNames[kind],
    enabled: true,
    mode: 'inherit',
    slot_status: 'missing',
    config: {},
  };
}

export function WorkspaceSlotsPanel({ workspaceId, isAdmin }: Props) {
  const { t } = useTranslation();
  const [items, setItems] = useState<IntegrationSlot[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [editor, setEditor] = useState<EditorState | null>(null);
  const [saving, setSaving] = useState(false);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);

  const loadSlots = useCallback(async () => {
    setLoading(true);
    setLoadError(false);
    try {
      const response = await fetch('/api/v1/integrations', { credentials: 'include' });
      if (!response.ok) {
        throw new Error('load failed');
      }
      const data = (await response.json()) as { items?: IntegrationSlot[] };
      setItems((data.items ?? []).filter((item) => item.workspace_id === workspaceId));
    } catch {
      setItems([]);
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  }, [workspaceId]);

  useEffect(() => {
    void loadSlots();
  }, [loadSlots]);

  const slots = useMemo(
    () => kinds.map((kind) => items.find((item) => item.kind === kind) ?? placeholderSlot(workspaceId, kind)),
    [items, workspaceId],
  );

  const openEditor = (slot: IntegrationSlot) => {
    setEditor({
      slot,
      mode: slot.mode,
      form: configFormFromItem(slot.kind, slot.config),
    });
  };

  const changeMode = (mode: IntegrationMode) => {
    setEditor((current) => {
      if (!current || current.mode === mode) {
        return current;
      }
      if (
        current.mode === 'custom' &&
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
      return { ...current, mode, form };
    });
  };

  const editorReady =
    editor !== null &&
    (editor.mode === 'inherit'
      ? editor.slot.kind !== 'jira' || editor.form.project_key.trim() !== ''
      : integrationFormReady(editor.slot.kind, editor.form, {
          workspaceOnly: false,
          editing: editor.slot.mode === 'custom',
        }));

  const saveEditor = async () => {
    if (!editor?.slot.id) {
      return;
    }
    setSaving(true);
    setToast(null);
    try {
      const config =
        editor.mode === 'inherit'
          ? editor.slot.kind === 'jira'
            ? { project_key: editor.form.project_key.trim() }
            : {}
          : buildConfigPayload(editor.slot.kind, editor.form, {
              workspaceOnly: false,
              keepBlankSecrets: editor.slot.mode === 'custom',
            });
      const response = await fetch(`/api/v1/integrations/${editor.slot.id}`, {
        method: 'PATCH',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          mode: editor.mode,
          enabled: editor.slot.enabled,
          config,
        }),
      });
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('workspaces.integrations.save_failed'));
      }
      setEditor(null);
      setToast({ message: t('workspaces.integrations.save_success'), variant: 'success' });
      await loadSlots();
    } catch (error) {
      setToast({
        message: error instanceof Error ? error.message : t('workspaces.integrations.save_failed'),
        variant: 'default',
      });
    } finally {
      setSaving(false);
    }
  };

  const statusLabel = (status: SlotStatus) => t(`workspaces.integrations.status_${status}`);

  return (
    <section className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold text-zinc-900">{t('workspaces.integrations.title')}</h2>
        <p className="text-sm text-zinc-600">{t('workspaces.integrations.subtitle')}</p>
      </div>
      {loadError ? <Banner variant="warning">{t('workspaces.integrations.load_error')}</Banner> : null}
      {loading ? (
        <p className="text-sm text-zinc-500" aria-live="polite">
          {t('integrations.loading')}
        </p>
      ) : null}
      <div aria-busy={loading}>
        <DataTable
          columns={[
            {
              key: 'kind',
              header: t('integrations.column.kind'),
              cellClassName: 'font-medium text-zinc-900',
              render: (slot) => kindNames[slot.kind],
            },
            {
              key: 'mode',
              header: t('workspaces.integrations.mode'),
              render: (slot) => (
                <StatusTag
                  variant="neutral"
                  label={t(`workspaces.integrations.mode_${slot.mode}`)}
                />
              ),
            },
            {
              key: 'status',
              header: t('integrations.column.status'),
              render: (slot) => (
                <StatusTag
                  variant={slot.slot_status === 'ready' || slot.slot_status === 'using_global' ? 'resolved' : 'neutral'}
                  label={statusLabel(slot.slot_status)}
                />
              ),
            },
            ...(isAdmin
              ? [
                  {
                    key: 'actions',
                    header: t('integrations.column.actions'),
                    render: (slot: IntegrationSlot) => (
                      <Button
                        variant="secondary"
                        disabled={!slot.id || loading}
                        onClick={() => openEditor(slot)}
                      >
                        {t('workspaces.integrations.configure')}
                      </Button>
                    ),
                  },
                ]
              : []),
          ]}
          rows={slots}
          rowKey={(slot) => slot.kind}
          emptyMessage=""
        />
      </div>

      {editor ? (
        <Modal
          title={t('workspaces.integrations.configure_title', { kind: kindNames[editor.slot.kind] })}
          open
          onClose={() => setEditor(null)}
          primaryLabel={t('integrations.save')}
          secondaryLabel={t('teams.cancel')}
          onPrimary={() => void saveEditor()}
          primaryDisabled={saving || !editorReady}
          primaryLoading={saving}
        >
          <Select
            id="workspace-integration-mode"
            label={t('workspaces.integrations.mode')}
            value={editor.mode}
            options={[
              { value: 'inherit', label: t('workspaces.integrations.mode_inherit') },
              { value: 'custom', label: t('workspaces.integrations.mode_custom') },
            ]}
            onChange={(value) => changeMode(value as IntegrationMode)}
          />
          <p className="text-sm text-zinc-600">
            {t(
              editor.mode === 'inherit'
                ? 'workspaces.integrations.inherit_help'
                : 'workspaces.integrations.custom_help',
            )}
          </p>
          <IntegrationConfigFields
            kind={editor.slot.kind}
            form={editor.form}
            mode={editor.mode}
            workspaceOnly
            editing={editor.slot.mode === 'custom'}
            onChange={(form) => setEditor((current) => (current ? { ...current, form } : current))}
          />
        </Modal>
      ) : null}

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </section>
  );
}
