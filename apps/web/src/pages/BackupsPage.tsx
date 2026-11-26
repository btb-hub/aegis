import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';
import { Banner } from '../components/ui/Banner';
import { Button } from '../components/ui/Button';
import { DataTable } from '../components/ui/DataTable';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { PageContent } from '../components/ui/PageContent';
import { PageHeader } from '../components/ui/PageHeader';
import { StatusTag, type StatusVariant } from '../components/ui/StatusTag';
import { Toast } from '../components/ui/Toast';
import { formatDateTime } from '../lib/formatDate';
import {
  backupErrorKey,
  BackupsApiError,
  fetchBackupStatus,
  reviewBackupCheck,
  type BackupCheckStatus,
  type BackupItem,
  type BackupStatus,
} from '../lib/backupsApi';

const MAX_COMMENT = 500;

const resultVariant: Record<BackupCheckStatus, StatusVariant> = {
  passed: 'resolved',
  failed: 'open',
  running: 'neutral',
};

export function BackupsPage() {
  const { t, i18n } = useTranslation();
  const { user } = useAuth();
  const canReview = user?.role === 'admin' || user?.role === 'member';

  const [status, setStatus] = useState<BackupStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [toast, setToast] = useState<{ message: string; variant: 'default' | 'success' } | null>(null);
  const [rejecting, setRejecting] = useState<BackupItem | null>(null);
  const [comment, setComment] = useState('');
  const [commentError, setCommentError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      setStatus(await fetchBackupStatus());
    } catch (error) {
      setStatus(null);
      setLoadError(backupErrorKey(error, 'backups.error.load'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const date = (value: string) => formatDateTime(new Date(value), i18n.language);

  async function submitReview(item: BackupItem, decision: 'confirm' | 'reject', text = '') {
    setBusyId(item.check.id);
    setToast(null);
    try {
      const review = await reviewBackupCheck(item.check.id, decision, text);
      setStatus((current) => current && {
        ...current,
        items: current.items.map((row) => (row.check.id === item.check.id ? { ...row, review } : row)),
      });
      setToast({ message: t(decision === 'confirm' ? 'backups.toast.confirmed' : 'backups.toast.rejected'), variant: 'success' });
      setRejecting(null);
    } catch (error) {
      const key = backupErrorKey(error, 'backups.error.review');
      if (decision === 'reject' && key === 'backups.error.comment_required') {
        setCommentError(t(key));
        return;
      }
      setRejecting(null);
      setToast({ message: t(key), variant: 'default' });
      if (error instanceof BackupsApiError && error.code === 'CONFLICT') {
        void load();
      }
    } finally {
      setBusyId(null);
    }
  }

  function openReject(item: BackupItem) {
    setComment('');
    setCommentError(null);
    setRejecting(item);
  }

  function confirmReject() {
    if (!rejecting) {
      return;
    }
    const text = comment.trim();
    if (!text) {
      setCommentError(t('backups.error.comment_required'));
      return;
    }
    if (text.length > MAX_COMMENT) {
      setCommentError(t('backups.error.comment_too_long', { max: MAX_COMMENT }));
      return;
    }
    void submitReview(rejecting, 'reject', text);
  }

  const flag = (value: boolean | null, yes: string, no: string) => {
    if (value === null) {
      return <span className="text-zinc-500">{t('backups.value.unknown')}</span>;
    }
    return <span className={value ? 'text-resolved' : 'font-medium text-severity-p1'}>{t(value ? yes : no)}</span>;
  };

  const reviewCell = (item: BackupItem) => {
    if (item.review) {
      const key = 'backups.review.by';
      return <div className="space-y-1">
        <StatusTag variant={item.review.decision === 'confirmed' ? 'resolved' : 'open'}
          label={t(item.review.decision === 'confirmed' ? 'backups.review.confirmed' : 'backups.review.rejected')} />
        <p className="text-sm text-zinc-700">{t(key, { name: item.review.reviewer.display_name || item.review.reviewer.email, date: date(item.review.created_at) })}</p>
        {item.review.comment ? <p className="break-words text-sm text-zinc-600">{item.review.comment}</p> : null}
      </div>;
    }
    if (item.check.status === 'running') {
      return <span className="text-sm text-zinc-500">{t('backups.review.after_check')}</span>;
    }
    return <StatusTag variant="acknowledged" label={t('backups.review.waiting')} />;
  };

  const actionsCell = (item: BackupItem) => {
    if (!canReview || item.review || item.check.status === 'running') {
      return null;
    }
    const busy = busyId === item.check.id;
    return <div className="flex flex-wrap gap-2">
      {item.check.status === 'passed'
        ? <Button disabled={busy} onClick={() => void submitReview(item, 'confirm')}>{t('backups.action.confirm')}</Button>
        : null}
      <Button variant="secondary" disabled={busy} onClick={() => openReject(item)}>{t('backups.action.reject')}</Button>
    </div>;
  };

  return (
    <PageContent>
      <PageHeader
        title={t('backups.page_title')}
        subtitle={t('backups.page_subtitle')}
        actions={<Button variant="secondary" disabled={loading} onClick={() => void load()}>{t('backups.action.refresh')}</Button>}
      />

      {loadError ? <div className="mb-4 space-y-2">
        <Banner variant="warning">{t(loadError)}</Banner>
        {loadError === 'backups.error.not_configured' && user?.role === 'admin'
          ? <Link className="text-sm text-accent hover:underline" to="/settings?section=backups">{t('backups.open_settings')}</Link>
          : null}
      </div> : null}

      {loading && !status ? <p role="status" className="text-sm text-zinc-600">{t('backups.loading')}</p> : null}

      {status ? <div className="space-y-3">
        <p className="text-sm text-zinc-600">{t('backups.stale_hint', { hours: status.stale_after_hours })}</p>
        <DataTable
          stackOnMobile
          columns={[
            {
              key: 'backup',
              header: t('backups.col.backup'),
              cellClassName: 'font-medium text-zinc-900',
              render: (item) => <div className="space-y-1">
                <span>{item.backup}</span>
                {item.stale ? <StatusTag variant="acknowledged" label={t('backups.stale', { hours: status.stale_after_hours })} /> : null}
              </div>,
            },
            {
              key: 'result',
              header: t('backups.col.result'),
              render: (item) => <div className="space-y-1">
                <StatusTag variant={resultVariant[item.check.status]} label={t(`backups.result.${item.check.status}`)} />
                {item.check.message ? <p className="break-words text-sm text-zinc-600">{item.check.message}</p> : null}
              </div>,
            },
            {
              key: 'checked',
              header: t('backups.col.checked_at'),
              cellClassName: 'text-zinc-700',
              render: (item) => date(item.check.finished_at ?? item.check.started_at),
            },
            { key: 'replicas', header: t('backups.col.replicas'), render: (item) => flag(item.check.replicas_match, 'backups.value.match', 'backups.value.mismatch') },
            { key: 'restore', header: t('backups.col.restore'), render: (item) => flag(item.check.restore_ok, 'backups.value.restored', 'backups.value.restore_failed') },
            { key: 'manifest', header: t('backups.col.manifest'), render: (item) => flag(item.check.manifest_counts_match, 'backups.value.match', 'backups.value.mismatch') },
            { key: 'review', header: t('backups.col.review'), render: reviewCell },
            { key: 'actions', header: t('backups.col.actions'), render: actionsCell },
          ]}
          rows={status.items}
          rowKey={(item) => item.check.id}
          emptyMessage={t('backups.empty')}
        />
      </div> : null}

      <Modal
        open={rejecting !== null}
        title={t('backups.reject_title', { backup: rejecting?.backup ?? '' })}
        onClose={() => setRejecting(null)}
        primaryLabel={t('backups.action.reject')}
        secondaryLabel={t('backups.action.cancel')}
        primaryLoading={rejecting !== null && busyId === rejecting.check.id}
        onPrimary={confirmReject}
      >
        <p className="text-sm text-zinc-700">{t('backups.reject_body')}</p>
        <Input label={t('backups.comment_label')} value={comment} error={commentError ?? undefined}
          hint={t('backups.comment_hint', { max: MAX_COMMENT })}
          onChange={(value) => { setComment(value); setCommentError(null); }} />
      </Modal>

      {toast ? <Toast message={toast.message} variant={toast.variant} /> : null}
    </PageContent>
  );
}
