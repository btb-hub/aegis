import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import type { Team } from '../lib/teamTypes';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { PageBreadcrumb } from '../components/ui/PageBreadcrumb';
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
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <PageBreadcrumb
            ariaLabel={t('nav.breadcrumb_label')}
            items={[{ label: t('nav.teams') }]}
          />
          <h1 className="text-2xl font-semibold text-zinc-900">{t('teams.page_title')}</h1>
          <p className="mt-1 text-sm text-zinc-600">{t('teams.page_subtitle')}</p>
        </div>
        {isAdmin ? (
          <Button onClick={openCreate}>{t('teams.create')}</Button>
        ) : null}
      </div>

      {loadError ? (
        <div
          role="alert"
          className="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950"
        >
          {loadError}
        </div>
      ) : null}

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
        <div className="overflow-hidden rounded-lg border border-zinc-200 bg-white">
          <table className="min-w-full divide-y divide-zinc-200 text-sm">
            <thead className="bg-zinc-50 text-left text-zinc-600">
              <tr>
                <th className="px-4 py-3 font-medium">{t('teams.column.name')}</th>
                <th className="px-4 py-3 font-medium">{t('teams.column.description')}</th>
                <th className="px-4 py-3 font-medium">{t('teams.column.actions')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200">
              {items.map((team) => (
                <tr key={team.id}>
                  <td className="px-4 py-3 font-medium text-zinc-900">
                    <Link to={`/teams/${team.id}`} className="text-accent hover:underline">
                      {team.name}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-zinc-700">{team.description || '—'}</td>
                  <td className="px-4 py-3">
                    {isAdmin ? (
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
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
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
    </div>
  );
}
