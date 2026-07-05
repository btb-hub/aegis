import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import type { Team } from '../lib/teamTypes';
import { Banner } from '../components/ui/Banner';
import { Button } from '../components/ui/Button';
import { DataTable } from '../components/ui/DataTable';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { Toast } from '../components/ui/Toast';

type TeamFormState = {
  name: string;
  description: string;
};

const emptyForm: TeamFormState = { name: '', description: '' };

export function TeamsPage() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [items, setItems] = useState<Team[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);

  const [formOpen, setFormOpen] = useState(false);
  const [editingTeam, setEditingTeam] = useState<Team | null>(null);
  const [form, setForm] = useState<TeamFormState>(emptyForm);
  const [saving, setSaving] = useState(false);

  const [deleteTarget, setDeleteTarget] = useState<Team | null>(null);
  const [deleting, setDeleting] = useState(false);

  const loadTeams = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const response = await fetch('/api/v1/teams', { credentials: 'include' });
      if (response.status === 401) {
        setLoadError(t('teams.sign_in_required'));
        setItems([]);
        return;
      }
      if (!response.ok) {
        throw new Error(t('teams.load_error'));
      }
      const data = (await response.json()) as { items: Team[] };
      setItems(data.items ?? []);
    } catch {
      setLoadError(t('teams.load_error'));
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void loadTeams();
  }, [loadTeams]);

  const openCreate = () => {
    setEditingTeam(null);
    setForm(emptyForm);
    setFormOpen(true);
  };

  const openEdit = (team: Team) => {
    setEditingTeam(team);
    setForm({ name: team.name, description: team.description ?? '' });
    setFormOpen(true);
  };

  const closeForm = () => {
    setFormOpen(false);
    setEditingTeam(null);
    setForm(emptyForm);
  };

  const saveTeam = async () => {
    setSaving(true);
    setToast(null);
    try {
      const response = await fetch(
        editingTeam ? `/api/v1/teams/${editingTeam.id}` : '/api/v1/teams',
        {
          method: editingTeam ? 'PATCH' : 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: form.name.trim(), description: form.description.trim() }),
        },
      );
      if (response.status === 401) {
        setToast({ message: t('teams.sign_in_required'), variant: 'default' });
        return;
      }
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('teams.save_failed'));
      }
      setToast({
        message: t(editingTeam ? 'teams.update_success' : 'teams.create_success'),
        variant: 'success',
      });
      closeForm();
      await loadTeams();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('teams.save_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setSaving(false);
    }
  };

  const confirmDelete = async () => {
    if (!deleteTarget) {
      return;
    }
    setDeleting(true);
    setToast(null);
    try {
      const response = await fetch(`/api/v1/teams/${deleteTarget.id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (response.status === 401) {
        setToast({ message: t('teams.sign_in_required'), variant: 'default' });
        return;
      }
      if (!response.ok) {
        const body = (await response.json()) as { message?: string };
        throw new Error(body.message ?? t('teams.delete_failed'));
      }
      setToast({ message: t('teams.delete_success'), variant: 'success' });
      setDeleteTarget(null);
      await loadTeams();
    } catch (error) {
      const message = error instanceof Error ? error.message : t('teams.delete_failed');
      setToast({ message, variant: 'default' });
    } finally {
      setDeleting(false);
    }
  };

  return (
    <PageContent>
      <PageHeader
        title={t('teams.page_title')}
        subtitle={t('teams.page_subtitle')}
        breadcrumb={{
          ariaLabel: t('nav.breadcrumb_label'),
          items: [{ label: t('nav.platform'), href: '/dashboard' }, { label: t('nav.teams') }],
        }}
        actions={isAdmin ? <Button onClick={openCreate}>{t('teams.create')}</Button> : undefined}
      />

      {loadError ? <Banner variant="warning">{loadError}</Banner> : null}

      {loading ? (
        <p className="text-sm text-zinc-600">{t('teams.loading')}</p>
      ) : loadError ? null : items.length === 0 ? (
        <div className="rounded-lg border border-dashed border-zinc-300 bg-zinc-50 px-6 py-10 text-center">
          <p className="text-sm text-zinc-600">{t('teams.empty')}</p>
          {isAdmin ? (
            <div className="mt-4">
              <Button onClick={openCreate}>{t('teams.empty_cta')}</Button>
            </div>
          ) : null}
        </div>
      ) : (
        <DataTable
          columns={[
            {
              key: 'name',
              header: t('teams.column.name'),
              cellClassName: 'font-medium text-zinc-900',
              render: (team) => (
                <Link to={`/teams/${team.id}`} className="text-accent hover:underline">
                  {team.name}
                </Link>
              ),
            },
            {
              key: 'description',
              header: t('teams.column.description'),
              cellClassName: 'text-zinc-700',
              render: (team) => team.description || '—',
            },
            {
              key: 'actions',
              header: t('teams.column.actions'),
              render: (team) =>
                isAdmin ? (
                  <div className="flex flex-wrap gap-2">
                    <Button variant="ghost" onClick={() => openEdit(team)}>
                      {t('teams.edit')}
                    </Button>
                    <Button variant="ghost" onClick={() => setDeleteTarget(team)}>
                      {t('teams.delete')}
                    </Button>
                  </div>
                ) : (
                  '—'
                ),
            },
          ]}
          rows={items}
          rowKey={(team) => team.id}
          emptyMessage={t('teams.empty')}
        />
      )}

      <Modal
        title={t(editingTeam ? 'teams.edit' : 'teams.create')}
        open={formOpen}
        onClose={closeForm}
        primaryLabel={t('teams.save')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void saveTeam()}
        primaryDisabled={!form.name.trim() || saving}
        primaryLoading={saving}
      >
        <Input label={t('teams.name_label')} value={form.name} onChange={(value) => setForm((f) => ({ ...f, name: value }))} />
        <Input
          label={t('teams.description_label')}
          value={form.description}
          onChange={(value) => setForm((f) => ({ ...f, description: value }))}
        />
      </Modal>

      <Modal
        title={t('teams.delete_confirm_title')}
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        primaryLabel={t('teams.delete')}
        secondaryLabel={t('teams.cancel')}
        onPrimary={() => void confirmDelete()}
        primaryDisabled={deleting}
        primaryLoading={deleting}
      >
        <p className="text-sm text-zinc-700">{t('teams.delete_confirm_body')}</p>
      </Modal>

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
