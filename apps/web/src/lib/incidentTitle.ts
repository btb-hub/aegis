export type IncidentTitleParts = {
  summary: string;
  context?: string;
};

export function splitIncidentTitle(title: string): IncidentTitleParts {
  const normalized = title.replace(/\s+/g, ' ').trim();
  const contextMarker = normalized.match(/\s+(?:where:|alertname:)\s*/i);

  if (!contextMarker || contextMarker.index === undefined || contextMarker.index === 0) {
    return { summary: normalized };
  }

  return {
    summary: normalized.slice(0, contextMarker.index).trim(),
    context: normalized.slice(contextMarker.index).trim(),
  };
}
