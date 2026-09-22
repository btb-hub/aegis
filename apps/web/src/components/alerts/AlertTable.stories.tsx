import type { Meta, StoryObj } from '@storybook/react';
import type { AlertItem } from '../../lib/alertTypes';
import { AlertTable } from './AlertTable';

const longAlertTitle =
  'HelmRelease is not in the Ready state (10.133.27.95:8080). ' +
  'HelmRelease amateur-teams-uniform in flux-system is not Ready (ready=Unknown) > 20 min. ' +
  'reason=Progressing Where: FluxHelmReleaseNotReady instance: 10.133.27.95:8080 ' +
  'job: kube-state-metrics namespace: flux-system exported_namespace: flux-system severity: critical ' +
  'priority: P2 service: kps-kube-state-metrics pod: kps-kube-state-metrics-7487977856-wbfv5 ' +
  'generatorURL: /graph?g0.expr=gotk_resource_info%7Bcustomresource_group%3D%22helm.toolkit.fluxcd.io%22%7D';

const alerts: AlertItem[] = [
  {
    id: 'alert-1',
    fingerprint: 'fp-1',
    status: 'firing',
    severity: 'critical',
    title: longAlertTitle,
    labels: {},
    received_at: '2026-09-22T07:57:15.269Z',
    incident_id: null,
  },
  {
    id: 'alert-2',
    fingerprint: 'fp-2',
    status: 'resolved',
    severity: 'warning',
    title: 'API error rate is above the service objective for checkout-api',
    labels: {},
    received_at: '2026-09-22T07:40:15.269Z',
    incident_id: null,
  },
];

const meta: Meta<typeof AlertTable> = {
  title: 'Alerts/AlertTable',
  component: AlertTable,
  tags: ['autodocs'],
  args: {
    items: alerts,
    total: alerts.length,
    page: 1,
    pageSize: 25,
    onPageChange: () => undefined,
  },
};

export default meta;
type Story = StoryObj<typeof AlertTable>;

export const LongAlertTitle: Story = {};

export const LongAlertTitleRussian: Story = {
  globals: { locale: 'ru' },
};
