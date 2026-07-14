import { useTranslation } from 'react-i18next';
import { Input } from '../ui/Input';

export type IntegrationKind = 'jira' | 'slack' | 'express';
export type IntegrationMode = 'inherit' | 'custom';

export type IntegrationConfigForm = {
  base_url: string;
  email: string;
  api_token: string;
  project_key: string;
  bot_token: string;
  signing_secret: string;
  bot_id: string;
  host: string;
  secret_key: string;
};

export const emptyIntegrationConfigForm = (): IntegrationConfigForm => ({
  base_url: '',
  email: '',
  api_token: '',
  project_key: '',
  bot_token: '',
  signing_secret: '',
  bot_id: '',
  host: '',
  secret_key: '',
});

export function configFormFromItem(
  kind: string,
  config: Record<string, unknown> | undefined,
): IntegrationConfigForm {
  const form = emptyIntegrationConfigForm();
  if (!config) {
    return form;
  }
  const read = (key: keyof IntegrationConfigForm) => {
    const value = config[key];
    return typeof value === 'string' && value !== '***' ? value : '';
  };
  if (kind === 'jira') {
    form.base_url = read('base_url');
    form.email = read('email');
    form.project_key = read('project_key');
  }
  if (kind === 'slack') {
    // secrets stay blank for edit
  }
  if (kind === 'express') {
    form.bot_id = read('bot_id');
    form.host = read('host');
  }
  return form;
}

/** Build POST/PATCH config; omit blank secrets so PATCH keeps stored values. */
export function buildConfigPayload(
  kind: IntegrationKind,
  form: IntegrationConfigForm,
  opts: { workspaceOnly: boolean; keepBlankSecrets: boolean },
): Record<string, string> {
  if (opts.workspaceOnly) {
    return { project_key: form.project_key.trim() };
  }
  if (kind === 'jira') {
    const out: Record<string, string> = {
      base_url: form.base_url.trim(),
      email: form.email.trim(),
      project_key: form.project_key.trim(),
    };
    const token = form.api_token.trim();
    if (token || !opts.keepBlankSecrets) {
      out.api_token = token;
    }
    return out;
  }
  if (kind === 'slack') {
    const out: Record<string, string> = {};
    const bot = form.bot_token.trim();
    const signing = form.signing_secret.trim();
    if (bot || !opts.keepBlankSecrets) {
      out.bot_token = bot;
    }
    if (signing || !opts.keepBlankSecrets) {
      out.signing_secret = signing;
    }
    return out;
  }
  const out: Record<string, string> = {
    bot_id: form.bot_id.trim(),
    host: form.host.trim(),
  };
  const secret = form.secret_key.trim();
  if (secret || !opts.keepBlankSecrets) {
    out.secret_key = secret;
  }
  return out;
}

export function integrationFormReady(
  kind: IntegrationKind,
  form: IntegrationConfigForm,
  opts: { workspaceOnly: boolean; editing: boolean },
): boolean {
  if (opts.workspaceOnly) {
    return form.project_key.trim() !== '';
  }
  if (kind === 'jira') {
    const secretsOk = opts.editing || form.api_token.trim() !== '';
    return (
      form.base_url.trim() !== '' &&
      form.email.trim() !== '' &&
      form.project_key.trim() !== '' &&
      secretsOk
    );
  }
  if (kind === 'slack') {
    if (opts.editing) {
      return true;
    }
    return form.bot_token.trim() !== '' && form.signing_secret.trim() !== '';
  }
  const secretOk = opts.editing || form.secret_key.trim() !== '';
  return form.bot_id.trim() !== '' && form.host.trim() !== '' && secretOk;
}

type Props = {
  kind: IntegrationKind;
  form: IntegrationConfigForm;
  onChange: (next: IntegrationConfigForm) => void;
  workspaceOnly: boolean;
  editing: boolean;
  mode?: IntegrationMode;
};

export function IntegrationConfigFields({ kind, form, onChange, workspaceOnly, editing, mode }: Props) {
  const { t } = useTranslation();
  const secretHint = editing ? t('integrations.secret_keep_hint') : undefined;
  const inheritMode = mode === 'inherit' || (mode === undefined && workspaceOnly);

  if (inheritMode && kind === 'jira') {
    return (
      <Input
        label={t('setup.integrations.jira.project_key')}
        value={form.project_key}
        onChange={(value) => onChange({ ...form, project_key: value })}
      />
    );
  }
  if (inheritMode) {
    return null;
  }

  if (kind === 'jira') {
    return (
      <>
        <Input
          label={t('setup.integrations.jira.base_url')}
          value={form.base_url}
          onChange={(value) => onChange({ ...form, base_url: value })}
        />
        <Input
          label={t('setup.integrations.jira.email')}
          value={form.email}
          onChange={(value) => onChange({ ...form, email: value })}
          autoComplete="off"
        />
        <Input
          label={t('setup.integrations.jira.api_token')}
          value={form.api_token}
          onChange={(value) => onChange({ ...form, api_token: value })}
          type="password"
          autoComplete="new-password"
          hint={secretHint}
        />
        <Input
          label={t('setup.integrations.jira.project_key')}
          value={form.project_key}
          onChange={(value) => onChange({ ...form, project_key: value })}
        />
      </>
    );
  }

  if (kind === 'slack') {
    return (
      <>
        <Input
          label={t('setup.integrations.slack.bot_token')}
          value={form.bot_token}
          onChange={(value) => onChange({ ...form, bot_token: value })}
          type="password"
          autoComplete="new-password"
          hint={secretHint}
        />
        <Input
          label={t('setup.integrations.slack.signing_secret')}
          value={form.signing_secret}
          onChange={(value) => onChange({ ...form, signing_secret: value })}
          type="password"
          autoComplete="new-password"
          hint={secretHint}
        />
      </>
    );
  }

  return (
    <>
      <Input
        label={t('setup.integrations.express.bot_id')}
        value={form.bot_id}
        onChange={(value) => onChange({ ...form, bot_id: value })}
      />
      <Input
        label={t('setup.integrations.express.host')}
        value={form.host}
        onChange={(value) => onChange({ ...form, host: value })}
      />
      <Input
        label={t('setup.integrations.express.secret_key')}
        value={form.secret_key}
        onChange={(value) => onChange({ ...form, secret_key: value })}
        type="password"
        autoComplete="new-password"
        hint={secretHint}
      />
    </>
  );
}
