import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../components/ui/Button';
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
  const [testingId, setTestingId] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);

  const loadIntegrations = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/integrations', { credentials: 'include' });
      if (!response.ok) {
        throw new Error(t('integrations.load_error'));
      }
      const data = (await response.json()) as { items: IntegrationItem[] };
      setItems(data.items ?? []);
    } catch {
      setToast({ message: t('integrations.load_error'), variant: 'default' });
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void loadIntegrations();
  }, [loadIntegrations]);

  const testConnection = async (id: string) => {
    setTestingId(id);
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

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-zinc-900">{t('integrations.page_title')}</h1>
        <p className="mt-1 text-sm text-zinc-600">{t('integrations.page_subtitle')}</p>
      </div>

      {loading ? (
        <p className="text-sm text-zinc-600">{t('integrations.loading')}</p>
      ) : items.length === 0 ? (
        <p className="text-sm text-zinc-600">{t('integrations.empty')}</p>
      ) : (
        <div className="overflow-hidden rounded-lg border border-zinc-200 bg-white">
          <table className="min-w-full divide-y divide-zinc-200 text-sm">
            <thead className="bg-zinc-50 text-left text-zinc-600">
              <tr>
                <th className="px-4 py-3 font-medium">{t('integrations.column.name')}</th>
                <th className="px-4 py-3 font-medium">{t('integrations.column.kind')}</th>
                <th className="px-4 py-3 font-medium">{t('integrations.column.status')}</th>
                <th className="px-4 py-3 font-medium">{t('integrations.column.actions')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200">
              {items.map((item) => (
                <tr key={item.id}>
                  <td className="px-4 py-3 font-medium text-zinc-900">{item.name}</td>
                  <td className="px-4 py-3 text-zinc-700">{item.kind}</td>
                  <td className="px-4 py-3 text-zinc-700">
                    {item.enabled ? t('integrations.enabled') : t('integrations.disabled')}
                  </td>
                  <td className="px-4 py-3">
                    <Button
                      variant="secondary"
                      disabled={testingId === item.id}
                      onClick={() => void testConnection(item.id)}
                    >
                      {testingId === item.id ? t('integrations.testing') : t('integrations.test_connection')}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </div>
  );
}
