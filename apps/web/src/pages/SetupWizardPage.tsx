import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Toast } from '../components/ui/Toast';
import { DEFAULT_WORKSPACE_ID } from '../lib/teamTypes';
import { addEscalationPath, fetchWorkspaces } from '../lib/workspacesApi';
import { loadSetupWizardState, saveSetupWizardState } from '../lib/setupWizard';

type IntegrationItem = {
  id: string;
  kind: string;
  name: string;
  enabled: boolean;
};

const STEP_COUNT = 6;

export function SetupWizardPage() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const [step, setStep] = useState(() => loadSetupWizardState().step);
  const [healthOk, setHealthOk] = useState<boolean | null>(null);
  const [integrations, setIntegrations] = useState<IntegrationItem[]>([]);
  const [savingKind, setSavingKind] = useState<string | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [alertId, setAlertId] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);

  const [jiraConfig, setJiraConfig] = useState({
    base_url: '',
    email: '',
    api_token: '',
    project_key: '',
  });
  const [slackConfig, setSlackConfig] = useState({ bot_token: '', signing_secret: '' });
  const [expressConfig, setExpressConfig] = useState({ bot_id: '', host: '', secret_key: '' });

  const stepLabels = useMemo(
    () => [
      t('setup.steps.welcome'),
      t('setup.steps.auth'),
      t('setup.steps.team'),
      t('setup.steps.integrations'),
      t('setup.steps.test_alert'),
      t('setup.steps.done'),
    ],
    [t],
  );

  const [teamName, setTeamName] = useState('');
  const [teamDescription, setTeamDescription] = useState('');
  const [workspaceName, setWorkspaceName] = useState('');
  const [l2TeamName, setL2TeamName] = useState('');
  const [l3TeamName, setL3TeamName] = useState('');
  const [createdTeamId, setCreatedTeamId] = useState<string | null>(null);
  const [createdWorkspaceId, setCreatedWorkspaceId] = useState<string | null>(null);
  const [creatingTeam, setCreatingTeam] = useState(false);
  const [workspaceStepSkippable, setWorkspaceStepSkippable] = useState(false);

  useEffect(() => {
    void fetchWorkspaces()
      .then((items) => {
        const hasConfiguredWorkspace = items.some(
          (item) => item.id !== DEFAULT_WORKSPACE_ID && item.team_count > 0,
        );
        setWorkspaceStepSkippable(hasConfiguredWorkspace);
      })
      .catch(() => {
        setWorkspaceStepSkippable(false);
      });
  }, []);

  useEffect(() => {
    saveSetupWizardState({ step, completed: step >= STEP_COUNT - 1 });
  }, [step]);

  const checkHealth = useCallback(async () => {
    try {
      const response = await fetch('/healthz');
      setHealthOk(response.ok);
    } catch {
      setHealthOk(false);
    }
  }, []);

  const loadIntegrations = useCallback(async () => {
    const response = await fetch('/api/v1/integrations', { credentials: 'include' });
    if (!response.ok) {
      return;
    }
    const data = (await response.json()) as { items: IntegrationItem[] };
    setIntegrations(data.items ?? []);
  }, []);

  useEffect(() => {
    void checkHealth();
    void loadIntegrations();
  }, [checkHealth, loadIntegrations]);

  const upsertIntegration = async (kind: string, name: string, config: Record<string, string>) => {
    setSavingKind(kind);
    setToast(null);
    try {
      const response = await fetch('/api/v1/integrations', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ kind, name, config, enabled: true }),
      });
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('setup.integrations.save_failed'));
      }
      setToast({ message: t('setup.integrations.save_success', { kind }), variant: 'success' });
      await loadIntegrations();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('setup.integrations.save_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setSavingKind(null);
    }
  };

  const testIntegration = async (id: string) => {
    setTestingId(id);
    setToast(null);
    try {
      const response = await fetch(`/api/v1/integrations/${id}/test`, {
        method: 'POST',
        credentials: 'include',
      });
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('integrations.test_failed'));
      }
      setToast({ message: t('integrations.test_success'), variant: 'success' });
    } catch (error) {
      const message = error instanceof Error ? error.message : t('integrations.test_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setTestingId(null);
    }
  };

  const sendTestAlert = async () => {
    setToast(null);
    try {
      const response = await fetch('/api/v1/setup/test-alert', {
        method: 'POST',
        credentials: 'include',
      });
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('setup.test_alert.failed'));
      }
      const data = (await response.json()) as { id: string };
      setAlertId(data.id);
      setToast({ message: t('setup.test_alert.success', { id: data.id }), variant: 'success' });
    } catch (error) {
      const message = error instanceof Error ? error.message : t('setup.test_alert.failed');
      setToast({ message, variant: 'default' });
    }
  };

  const createTeam = async () => {
    if (!teamName.trim()) {
      return;
    }
    setCreatingTeam(true);
    setToast(null);
    try {
      const response = await fetch('/api/v1/teams', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          workspace_id: DEFAULT_WORKSPACE_ID,
          name: teamName.trim(),
          description: teamDescription.trim(),
        }),
      });
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('setup.team.create_failed'));
      }
      const data = (await response.json()) as { id: string; name: string };
      setCreatedTeamId(data.id);
      setToast({ message: t('setup.team.created', { name: data.name }), variant: 'success' });
    } catch (error) {
      const message = error instanceof Error ? error.message : t('setup.team.create_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setCreatingTeam(false);
    }
  };

  const createWorkspaceStructure = async () => {
    if (!l2TeamName.trim() || !l3TeamName.trim()) {
      return;
    }
    setCreatingTeam(true);
    setToast(null);
    try {
      let workspaceId = createdWorkspaceId ?? DEFAULT_WORKSPACE_ID;
      if (workspaceName.trim()) {
        const workspaceResponse = await fetch('/api/v1/workspaces', {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: workspaceName.trim() }),
        });
        if (!workspaceResponse.ok) {
          const body = (await workspaceResponse.json()) as { message?: string };
          throw new Error(body.message ?? t('setup.workspace.create_failed'));
        }
        const workspace = (await workspaceResponse.json()) as { id: string; name: string };
        workspaceId = workspace.id;
        setCreatedWorkspaceId(workspace.id);
      } else {
        const existing = await fetchWorkspaces();
        const defaultWorkspace = existing.find((item) => item.id === DEFAULT_WORKSPACE_ID) ?? existing[0];
        if (defaultWorkspace) {
          workspaceId = defaultWorkspace.id;
          setCreatedWorkspaceId(defaultWorkspace.id);
        }
      }

      const createTeamRequest = async (name: string, supportTier: string) => {
        const response = await fetch('/api/v1/teams', {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            workspace_id: workspaceId,
            name: name.trim(),
            support_tier: supportTier,
          }),
        });
        if (!response.ok) {
          const body = (await response.json()) as { message?: string };
          throw new Error(body.message ?? t('setup.team.create_failed'));
        }
        return (await response.json()) as { id: string; name: string };
      };

      const l2Team = await createTeamRequest(l2TeamName, 'l2');
      const l3Team = await createTeamRequest(l3TeamName, 'l3');
      await addEscalationPath(workspaceId, {
        from_team_id: l2Team.id,
        to_team_id: l3Team.id,
      });
      setCreatedTeamId(l2Team.id);
      setToast({
        message: t('setup.team.structure_created', { l2: l2Team.name, l3: l3Team.name }),
        variant: 'success',
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : t('setup.team.create_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setCreatingTeam(false);
    }
  };

  const integrationByKind = (kind: string) => integrations.find((item) => item.kind === kind);

  return (
    <PageContent className="mx-auto max-w-3xl">
      <PageHeader
        title={t('setup.page_title')}
        subtitle={t('setup.page_subtitle')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('nav.setup') }],
        }}
      />

      <nav aria-label={t('setup.progress_label')}>
        <ol className="flex flex-wrap gap-3 text-sm">
          {stepLabels.map((label, index) => (
            <li
              key={label}
              className={`rounded-full px-3 py-1 ${
                index === step ? 'bg-accent text-white' : 'bg-zinc-100 text-zinc-700'
              }`}
            >
              {index + 1}. {label}
            </li>
          ))}
        </ol>
      </nav>

      <section className="rounded-lg border border-zinc-200 bg-white p-6" aria-live="polite">
        {step === 0 ? (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-zinc-900">{t('setup.welcome.title')}</h2>
            <p className="text-sm text-zinc-600">{t('setup.welcome.body')}</p>
            <p className="text-sm text-zinc-700">
              {healthOk === null
                ? t('setup.welcome.checking')
                : healthOk
                  ? t('setup.welcome.health_ok')
                  : t('setup.welcome.health_failed')}
            </p>
            <Button variant="secondary" onClick={() => void checkHealth()}>
              {t('setup.welcome.retry_health')}
            </Button>
          </div>
        ) : null}

        {step === 1 ? (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-zinc-900">{t('setup.auth.title')}</h2>
            <p className="text-sm text-zinc-600">{t('setup.auth.body')}</p>
            {user ? (
              <p className="text-sm text-zinc-900">{t('setup.auth.signed_in', { name: user.display_name || user.email })}</p>
            ) : (
              <p className="text-sm text-amber-800">{t('setup.auth.sign_in_required')}</p>
            )}
            <Link className="text-sm text-accent hover:underline" to="/login">
              {t('setup.auth.open_login')}
            </Link>
            {user ? (
              <Link className="block text-sm text-accent hover:underline" to="/account">
                {t('setup.team.review_account')}
              </Link>
            ) : null}
          </div>
        ) : null}

        {step === 2 ? (
          <div className="space-y-6">
            <div className="space-y-4">
              <h2 className="text-lg font-semibold text-zinc-900">{t('setup.workspace.title')}</h2>
              <p className="text-sm text-zinc-600">{t('setup.workspace.body_manage')}</p>
              <Link className="inline-flex text-sm text-accent hover:underline" to="/workspaces">
                {t('setup.workspace.open_workspaces')}
              </Link>
              {workspaceStepSkippable ? (
                <p className="text-sm text-zinc-600">{t('setup.workspace.skip_hint')}</p>
              ) : null}
            </div>

            <div className="space-y-4 rounded-md border border-dashed border-zinc-200 p-4">
              <h3 className="font-medium text-zinc-900">{t('setup.workspace.quick_title')}</h3>
              <p className="text-sm text-zinc-600">{t('setup.workspace.quick_body')}</p>
              <Input
                label={t('setup.workspace.name_label')}
                value={workspaceName}
                onChange={setWorkspaceName}
              />
              <Input label={t('setup.team.l2_name_label')} value={l2TeamName} onChange={setL2TeamName} />
              <Input label={t('setup.team.l3_name_label')} value={l3TeamName} onChange={setL3TeamName} />
              <Button
                variant="secondary"
                disabled={creatingTeam || !l2TeamName.trim() || !l3TeamName.trim()}
                onClick={() => void createWorkspaceStructure()}
              >
                {t('setup.team.create_structure')}
              </Button>
              {createdTeamId ? (
                <Link className="text-sm text-accent hover:underline" to={`/teams/${createdTeamId}/shifts`}>
                  {t('setup.team.open_shifts')}
                </Link>
              ) : null}
            </div>

            <div className="space-y-4 rounded-md border border-zinc-200 p-4">
              <h3 className="font-medium text-zinc-900">{t('setup.team.title')}</h3>
              <p className="text-sm text-zinc-600">{t('setup.team.body')}</p>
              <Input label={t('setup.team.name_label')} value={teamName} onChange={setTeamName} />
              <Input label={t('setup.team.description_label')} value={teamDescription} onChange={setTeamDescription} />
              <Button disabled={creatingTeam || !teamName.trim()} onClick={() => void createTeam()}>
                {t('setup.team.create')}
              </Button>
            </div>
          </div>
        ) : null}

        {step === 3 ? (
          <div className="space-y-6">
            <h2 className="text-lg font-semibold text-zinc-900">{t('setup.integrations.title')}</h2>
            <p className="text-sm text-zinc-600">{t('setup.integrations.body')}</p>

            <div className="space-y-3 rounded-md border border-zinc-200 p-4">
              <h3 className="font-medium text-zinc-900">Jira</h3>
              <Input label={t('setup.integrations.jira.base_url')} value={jiraConfig.base_url} onChange={(value) => setJiraConfig((c) => ({ ...c, base_url: value }))} />
              <Input label={t('setup.integrations.jira.email')} value={jiraConfig.email} onChange={(value) => setJiraConfig((c) => ({ ...c, email: value }))} />
              <Input label={t('setup.integrations.jira.api_token')} value={jiraConfig.api_token} onChange={(value) => setJiraConfig((c) => ({ ...c, api_token: value }))} />
              <Input label={t('setup.integrations.jira.project_key')} value={jiraConfig.project_key} onChange={(value) => setJiraConfig((c) => ({ ...c, project_key: value }))} />
              <div className="flex gap-2">
                <Button variant="secondary" disabled={savingKind === 'jira'} onClick={() => void upsertIntegration('jira', 'Jira', jiraConfig)}>
                  {t('setup.integrations.save')}
                </Button>
                {integrationByKind('jira') ? (
                  <Button variant="secondary" disabled={testingId === integrationByKind('jira')?.id} onClick={() => void testIntegration(integrationByKind('jira')!.id)}>
                    {t('integrations.test_connection')}
                  </Button>
                ) : null}
              </div>
            </div>

            <div className="space-y-3 rounded-md border border-zinc-200 p-4">
              <h3 className="font-medium text-zinc-900">Slack</h3>
              <Input label={t('setup.integrations.slack.bot_token')} value={slackConfig.bot_token} onChange={(value) => setSlackConfig((c) => ({ ...c, bot_token: value }))} />
              <Input label={t('setup.integrations.slack.signing_secret')} value={slackConfig.signing_secret} onChange={(value) => setSlackConfig((c) => ({ ...c, signing_secret: value }))} />
              <div className="flex gap-2">
                <Button variant="secondary" disabled={savingKind === 'slack'} onClick={() => void upsertIntegration('slack', 'Slack', slackConfig)}>
                  {t('setup.integrations.save')}
                </Button>
                {integrationByKind('slack') ? (
                  <Button variant="secondary" disabled={testingId === integrationByKind('slack')?.id} onClick={() => void testIntegration(integrationByKind('slack')!.id)}>
                    {t('integrations.test_connection')}
                  </Button>
                ) : null}
              </div>
            </div>

            <div className="space-y-3 rounded-md border border-zinc-200 p-4">
              <h3 className="font-medium text-zinc-900">eXpress</h3>
              <Input label={t('setup.integrations.express.bot_id')} value={expressConfig.bot_id} onChange={(value) => setExpressConfig((c) => ({ ...c, bot_id: value }))} />
              <Input label={t('setup.integrations.express.host')} value={expressConfig.host} onChange={(value) => setExpressConfig((c) => ({ ...c, host: value }))} />
              <Input label={t('setup.integrations.express.secret_key')} value={expressConfig.secret_key} onChange={(value) => setExpressConfig((c) => ({ ...c, secret_key: value }))} />
              <div className="flex gap-2">
                <Button variant="secondary" disabled={savingKind === 'express'} onClick={() => void upsertIntegration('express', 'eXpress', expressConfig)}>
                  {t('setup.integrations.save')}
                </Button>
                {integrationByKind('express') ? (
                  <Button variant="secondary" disabled={testingId === integrationByKind('express')?.id} onClick={() => void testIntegration(integrationByKind('express')!.id)}>
                    {t('integrations.test_connection')}
                  </Button>
                ) : null}
              </div>
            </div>
          </div>
        ) : null}

        {step === 4 ? (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-zinc-900">{t('setup.test_alert.title')}</h2>
            <p className="text-sm text-zinc-600">{t('setup.test_alert.body')}</p>
            <Button onClick={() => void sendTestAlert()}>{t('setup.test_alert.send')}</Button>
            {alertId ? (
              <p className="text-sm text-zinc-700">{t('setup.test_alert.received', { id: alertId })}</p>
            ) : null}
          </div>
        ) : null}

        {step === 5 ? (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-zinc-900">{t('setup.done.title')}</h2>
            <p className="text-sm text-zinc-600">{t('setup.done.body')}</p>
            <Link className="text-sm text-accent hover:underline" to="/dashboard">
              {t('setup.done.open_dashboard')}
            </Link>
          </div>
        ) : null}
      </section>

      <div className="flex justify-between">
        <Button variant="secondary" disabled={step === 0} onClick={() => setStep((current) => Math.max(0, current - 1))}>
          {t('setup.back')}
        </Button>
        <div className="flex gap-2">
          {step === 2 && workspaceStepSkippable ? (
            <Button variant="secondary" onClick={() => setStep(3)}>
              {t('setup.workspace.skip_step')}
            </Button>
          ) : null}
          <Button disabled={step >= STEP_COUNT - 1} onClick={() => setStep((current) => Math.min(STEP_COUNT - 1, current + 1))}>
            {t('setup.next')}
          </Button>
        </div>
      </div>

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
