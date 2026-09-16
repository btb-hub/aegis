import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';
import { Banner } from '../components/ui/Banner';
import { DataTable } from '../components/ui/DataTable';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Toast } from '../components/ui/Toast';
import { UserRoleSelect } from '../components/users/UserRoleSelect';
import { UsersApiError } from '../lib/apiClient';
import { queryKeys } from '../lib/queryClient';
import { useLoader } from '../lib/useLoader';
import { fetchUsers, patchUserRole, type ListedUser, type UserRole } from '../lib/usersApi';

export function UsersPage() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const queryClient = useQueryClient();

  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);
  const [savingId, setSavingId] = useState<string | null>(null);

  const usersQuery = useLoader(queryKeys.users.list(), () => fetchUsers(), { enabled: isAdmin });
  const items = usersQuery.data?.items ?? [];
  const loadError = usersQuery.isError ? t('users.load_error') : null;

  const roleMut = useMutation({
    mutationFn: ({ id, role }: { id: string; role: UserRole }) => patchUserRole(id, role),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.users.all }),
  });

  const changeRole = async (targetUser: ListedUser, role: UserRole) => {
    setSavingId(targetUser.id);
    setToast(null);
    try {
      await roleMut.mutateAsync({ id: targetUser.id, role });
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

      {usersQuery.loading ? (
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
