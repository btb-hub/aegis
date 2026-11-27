export type BackupCheckStatus = 'running' | 'passed' | 'failed';

export type BackupCheck = {
  id: string;
  backup: string;
  engine: string;
  status: BackupCheckStatus;
  started_at: string;
  finished_at: string | null;
  replicas_match: boolean | null;
  restore_ok: boolean | null;
  manifest_counts_match: boolean | null;
  message?: string;
};

export type BackupReview = {
  check_id: string;
  decision: 'confirmed' | 'rejected';
  comment: string;
  reviewer: { id: string; display_name: string; email: string };
  created_at: string;
};

export type BackupItem = {
  backup: string;
  check: BackupCheck;
  stale: boolean;
  review: BackupReview | null;
};

export type BackupStatus = {
  stale_after_hours: number;
  items: BackupItem[];
};

export class BackupsApiError extends Error {
  status: number;
  code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, { credentials: 'include', ...init });
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as { code?: string; message?: string };
    throw new BackupsApiError(body.message ?? `request failed: ${response.status}`, response.status, body.code);
  }
  return (await response.json()) as T;
}

export function fetchBackupStatus(): Promise<BackupStatus> {
  return request<BackupStatus>('/api/v1/backups');
}

export function reviewBackupCheck(checkId: string, decision: 'confirm' | 'reject', comment = ''): Promise<BackupReview> {
  return request<BackupReview>(`/api/v1/backups/checks/${encodeURIComponent(checkId)}/${decision}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(decision === 'reject' ? { comment } : {}),
  });
}

/** Maps API error codes to locale keys; unknown errors fall back to a generic load message. */
export function backupErrorKey(error: unknown, fallback: string): string {
  if (!(error instanceof BackupsApiError)) {
    return fallback;
  }
  switch (error.code) {
    case 'BACKUP_CHECKER_NOT_CONFIGURED':
      return 'backups.error.not_configured';
    case 'BACKUP_CHECKER_UNAVAILABLE':
      return 'backups.error.unavailable';
    case 'BACKUP_CHECKER_INVALID_RESPONSE':
      return 'backups.error.invalid_response';
    case 'FORBIDDEN':
      return 'backups.error.forbidden';
    case 'NOT_FOUND':
      return 'backups.error.not_found';
    case 'CONFLICT':
      return 'backups.error.conflict';
    case 'VALIDATION_ERROR':
      return 'backups.error.comment_required';
    default:
      return fallback;
  }
}
