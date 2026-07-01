import type { Severity } from '../components/ui/SeverityTag';

export function severityToTag(severity: string): Severity {
  switch (severity.toLowerCase()) {
    case 'critical':
    case 'p1':
      return 'P1';
    case 'high':
    case 'warning':
    case 'p2':
      return 'P2';
    case 'moderate':
    case 'p3':
      return 'P3';
    case 'low':
    case 'info':
    case 'p4':
      return 'P4';
    default:
      return 'neutral';
  }
}

export function severityLabelKey(severity: string): string {
  return `incidents.severity.${severity.toLowerCase()}`;
}
