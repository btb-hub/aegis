import { describe, expect, it } from 'vitest';
import { formatDuration } from './analyticsTypes';
import { loadSetupWizardState, saveSetupWizardState } from './setupWizard';

describe('analyticsTypes', () => {
  it('formats short durations', () => {
    expect(formatDuration(45)).toBe('45s');
    expect(formatDuration(120)).toBe('2m');
    expect(formatDuration(7200)).toBe('2.0h');
  });
});

describe('setupWizard', () => {
  it('persists wizard step in localStorage', () => {
    saveSetupWizardState({ step: 2, completed: false });
    expect(loadSetupWizardState().step).toBe(2);
  });
});
