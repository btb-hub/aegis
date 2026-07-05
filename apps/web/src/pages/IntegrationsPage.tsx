import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Banner } from '../components/ui/Banner';
import { Button } from '../components/ui/Button';
import { DataTable } from '../components/ui/DataTable';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Toast } from '../components/ui/Toast';

type IntegrationItem = {
  id: string;
  kind: string;
  name: string;
  enabled: boolean;
};

export function IntegrationsPage() {
  const { t } = useTranslation();
  const [items, setItems] = useState<IntegrationItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);

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
      const message = error instanceof Error ? error.message : t('integrations.test_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setTestingId(null);
    }
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

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
