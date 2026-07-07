export type ApiErrorBody = {
  code?: string;
  message?: string;
};

const MESSAGE_KEYS: Record<string, string> = {
  'team name is required': 'errors.team_name_required',
  'workspace name is required': 'errors.workspace_name_required',
  'team_ids must not be empty': 'errors.team_ids_required',
  'team_ids must contain valid UUIDs': 'errors.team_ids_invalid',
  'workspace_id must be a valid uuid': 'errors.workspace_id_invalid',
};

export function resolveApiErrorMessage(
  t: (key: string) => string,
  body: ApiErrorBody,
  fallback: string,
): string {
  if (body.message && MESSAGE_KEYS[body.message]) {
    return t(MESSAGE_KEYS[body.message]);
  }
  return fallback;
}
