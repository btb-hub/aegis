import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';
import { Banner } from '../components/ui/Banner';
import { DataTable } from '../components/ui/DataTable';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Toast } from '../components/ui/Toast';
import { UserRoleSelect } from '../components/users/UserRoleSelect';
import { fetchUsers, patchUserRole, UsersApiError, type ListedUser, type UserRole } from '../lib/usersApi';

export function UsersPage() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [items, setItems] = useState<ListedUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);
  const [savingId, setSavingId] = useState<string | null>(null);

  const loadUsers = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const data = await fetchUsers();
      setItems(data.items ?? []);
    } catch {
      setLoadError(t('users.load_error'));
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    if (!isAdmin) {
      return;
    }
    void loadUsers();
  }, [isAdmin, loadUsers]);

  const changeRole = async (targetUser: ListedUser, role: UserRole) => {
    setSavingId(targetUser.id);
    setToast(null);
    try {
      const updated = await patchUserRole(targetUser.id, role);
      setItems((current) => current.map((item) => (item.id === updated.id ? { ...item, ...updated } : item)));
      setToast({ message: t('users.role_updated'), variant: 'success' });
    } catch (error) {
      const message = error instanceof UsersApiError ? error.message : t('users.load_error');
      setToast({ message, variant: 'default' });
    } finally {
      setSavingId(null);
    }
  };

  if (!isAdmin) {
    return (
      <PageContent>
        <PageHeader
          title={t('users.page_title')}
          subtitle={t('users.page_subtitle')}
          breadcrumb={{
            ariaLabel: t('nav.breadcrumb_label'),
            items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('nav.users') }],
          }}
        />
        <Banner variant="warning">{t('users.forbidden')}</Banner>
      </PageContent>
    );
  }

  return (
    <PageContent>
      <PageHeader
        title={t('users.page_title')}
        subtitle={t('users.page_subtitle')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('nav.users') }],
        }}
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}

      {loading ? (
        <p className="text-sm text-zinc-600">{t('users.loading')}</p>
      ) : loadError ? null : (
        <DataTable
          columns={[
            {
              key: 'name',
              header: t('users.col.name'),
              cellClassName: 'font-medium text-zinc-900',
              render: (item) => item.display_name,
            },
            {
              key: 'email',
              header: t('users.col.email'),
              cellClassName: 'text-zinc-700',
              render: (item) => item.email,
            },
            {
              key: 'role',
              header: t('users.col.role'),
              render: (item) => (
                <UserRoleSelect
                  id={`user-role-${item.id}`}
                  label={t('users.role_select_label', { name: item.display_name })}
                  hideLabel
                  value={item.role}
                  pinned={item.role_pinned}
                  disabled={savingId === item.id}
                  onChange={(role) => void changeRole(item, role)}
                />
              ),
            },
          ]}
          rows={items}
          rowKey={(item) => item.id}
          emptyMessage={t('users.empty')}
        />
      )}

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
