export type ApiErrorBody = {
  code?: string;
  message?: string;
  details?: Record<string, unknown> | null;
};

const MESSAGE_KEYS: Record<string, string> = {
  'team name is required': 'errors.team_name_required',
  'workspace name is required': 'errors.workspace_name_required',
  'team_ids must not be empty': 'errors.team_ids_required',
  'team_ids must contain valid UUIDs': 'errors.team_ids_invalid',
  'workspace_id must be a valid uuid': 'errors.workspace_id_invalid',
  'target team has no one on call': 'errors.handoff_no_on_call',
  'handoff target is not configured in escalation paths': 'errors.handoff_path_not_configured',
  'note is required': 'errors.handoff_note_required',
};

export function resolveApiErrorMessage(
  t: (key: string, options?: Record<string, unknown>) => string,
  body: ApiErrorBody,
  fallback: string,
): string {
  if (body.message && MESSAGE_KEYS[body.message]) {
    const key = MESSAGE_KEYS[body.message];
    if (body.message === 'target team has no one on call') {
      const teamName = body.details?.team_name;
      if (typeof teamName === 'string' && teamName.length > 0) {
        return t('errors.handoff_no_on_call_named', { team: teamName });
      }
    }
    return t(key);
  }
  if (body.message) {
    return body.message;
  }
  return fallback;
}
