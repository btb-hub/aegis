import { fireEvent, render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import {
  IntegrationConfigFields,
  buildConfigPayload,
  configFormFromItem,
  emptyIntegrationConfigForm,
  integrationFormReady,
} from './IntegrationConfigFields';

describe('IntegrationConfigFields helpers', () => {
  it('requires fields by kind on create', () => {
    const form = emptyIntegrationConfigForm();
    expect(integrationFormReady('jira', form, { workspaceOnly: false, editing: false })).toBe(false);
    expect(
      integrationFormReady(
        'jira',
        { ...form, base_url: 'https://jira', email: 'a@b.c', api_token: 't', project_key: 'OPS' },
        { workspaceOnly: false, editing: false },
      ),
    ).toBe(true);
    expect(integrationFormReady('slack', form, { workspaceOnly: false, editing: false })).toBe(false);
    expect(
      integrationFormReady(
        'slack',
        { ...form, bot_token: 'x', signing_secret: 'y' },
        { workspaceOnly: false, editing: false },
      ),
    ).toBe(true);
    expect(integrationFormReady('express', form, { workspaceOnly: false, editing: false })).toBe(false);
    expect(
      integrationFormReady(
        'express',
        { ...form, bot_id: 'bot', host: 'https://cts', secret_key: 's' },
        { workspaceOnly: false, editing: false },
      ),
    ).toBe(true);
    expect(integrationFormReady('jira', { ...form, project_key: 'OPS' }, { workspaceOnly: true, editing: false })).toBe(
      true,
    );
    expect(integrationFormReady('slack', form, { workspaceOnly: false, editing: true })).toBe(true);
  });

  it('omits blank secrets when keepBlankSecrets is true', () => {
    const form = {
      ...emptyIntegrationConfigForm(),
      base_url: 'https://jira',
      email: 'a@b.c',
      project_key: 'OPS',
      api_token: '',
    };
    expect(buildConfigPayload('jira', form, { workspaceOnly: false, keepBlankSecrets: true })).toEqual({
      base_url: 'https://jira',
      email: 'a@b.c',
      project_key: 'OPS',
    });
    expect(buildConfigPayload('slack', emptyIntegrationConfigForm(), { workspaceOnly: false, keepBlankSecrets: true })).toEqual(
      {},
    );
    expect(
      buildConfigPayload(
        'express',
        { ...emptyIntegrationConfigForm(), bot_id: 'bot', host: 'https://cts', secret_key: '' },
        { workspaceOnly: false, keepBlankSecrets: true },
      ),
    ).toEqual({ bot_id: 'bot', host: 'https://cts' });
    expect(
      buildConfigPayload('jira', { ...form, project_key: 'OPS' }, { workspaceOnly: true, keepBlankSecrets: false }),
    ).toEqual({ project_key: 'OPS' });
    expect(
      buildConfigPayload(
        'slack',
        { ...emptyIntegrationConfigForm(), bot_token: 'x', signing_secret: 'y' },
        { workspaceOnly: false, keepBlankSecrets: false },
      ),
    ).toEqual({ bot_token: 'x', signing_secret: 'y' });
  });

  it('prefills non-secret config and clears redacted secrets', () => {
    expect(
      configFormFromItem('jira', {
        base_url: 'https://jira',
        email: 'ops@example.com',
        api_token: '***',
        project_key: 'OPS',
      }),
    ).toEqual({
      ...emptyIntegrationConfigForm(),
      base_url: 'https://jira',
      email: 'ops@example.com',
      project_key: 'OPS',
    });
    expect(
      configFormFromItem('express', {
        bot_id: 'bot',
        host: 'https://cts',
        secret_key: '***',
      }),
    ).toEqual({
      ...emptyIntegrationConfigForm(),
      bot_id: 'bot',
      host: 'https://cts',
    });
    expect(configFormFromItem('slack', undefined)).toEqual(emptyIntegrationConfigForm());
  });

  it('renders kind-specific fields', () => {
    const form = emptyIntegrationConfigForm();
    const { unmount } = render(
      <I18nextProvider i18n={i18n}>
        <IntegrationConfigFields kind="jira" form={form} onChange={() => undefined} workspaceOnly={false} editing={false} />
      </I18nextProvider>,
    );
    expect(screen.getByText('Jira base URL')).toBeInTheDocument();
    unmount();

    render(
      <I18nextProvider i18n={i18n}>
        <IntegrationConfigFields kind="slack" form={form} onChange={() => undefined} workspaceOnly={false} editing />
      </I18nextProvider>,
    );
    expect(screen.getByText('Slack bot token')).toBeInTheDocument();
    expect(screen.getAllByText('Leave blank to keep the current value').length).toBeGreaterThan(0);
  });

  it('renders mode-specific workspace fields and reports field changes', () => {
    const form = emptyIntegrationConfigForm();
    const onChange = vi.fn();
    const { rerender } = render(
      <I18nextProvider i18n={i18n}>
        <IntegrationConfigFields
          kind="jira"
          form={form}
          onChange={onChange}
          workspaceOnly
          editing={false}
          mode="inherit"
        />
      </I18nextProvider>,
    );

    fireEvent.change(screen.getByLabelText('Jira project key'), { target: { value: 'OPS' } });
    expect(onChange).toHaveBeenCalledWith({ ...form, project_key: 'OPS' });

    rerender(
      <I18nextProvider i18n={i18n}>
        <IntegrationConfigFields
          kind="slack"
          form={form}
          onChange={onChange}
          workspaceOnly
          editing={false}
          mode="inherit"
        />
      </I18nextProvider>,
    );
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();

    rerender(
      <I18nextProvider i18n={i18n}>
        <IntegrationConfigFields
          kind="express"
          form={form}
          onChange={onChange}
          workspaceOnly
          editing={false}
          mode="custom"
        />
      </I18nextProvider>,
    );
    fireEvent.change(screen.getByLabelText('eXpress bot ID'), { target: { value: 'bot-1' } });
    fireEvent.change(screen.getByLabelText('eXpress bot host'), { target: { value: 'https://cts' } });
    fireEvent.change(screen.getByLabelText('eXpress secret key'), { target: { value: 'secret' } });
    expect(onChange).toHaveBeenCalledWith({ ...form, bot_id: 'bot-1' });
    expect(onChange).toHaveBeenCalledWith({ ...form, host: 'https://cts' });
    expect(onChange).toHaveBeenCalledWith({ ...form, secret_key: 'secret' });
  });
});
