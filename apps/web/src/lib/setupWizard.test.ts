import { afterEach, describe, expect, it, vi } from 'vitest';
import { loadSetupWizardState, saveSetupWizardState } from './setupWizard';

const STORAGE_KEY = 'aegis_setup_wizard';

describe('setupWizard state', () => {
  afterEach(() => {
    if (typeof localStorage !== 'undefined') {
      localStorage.removeItem(STORAGE_KEY);
    }
    vi.unstubAllGlobals();
  });

  it('returns defaults when nothing is stored', () => {
    expect(loadSetupWizardState()).toEqual({ step: 0, completed: false });
  });

  it('loads and normalizes stored state', () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ step: 2, completed: true }));
    expect(loadSetupWizardState()).toEqual({ step: 2, completed: true });
  });

  it('clamps step to valid range', () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ step: 99, completed: false }));
    expect(loadSetupWizardState().step).toBe(4);

    localStorage.setItem(STORAGE_KEY, JSON.stringify({ step: -3, completed: false }));
    expect(loadSetupWizardState().step).toBe(0);
  });

  it('returns defaults for invalid stored JSON', () => {
    localStorage.setItem(STORAGE_KEY, 'not-json');
    expect(loadSetupWizardState()).toEqual({ step: 0, completed: false });
  });

  it('returns defaults when step is not a number', () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ step: 'two', completed: true }));
    expect(loadSetupWizardState()).toEqual({ step: 0, completed: false });
  });

  it('persists state to localStorage', () => {
    saveSetupWizardState({ step: 3, completed: true });
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY)!)).toEqual({
      step: 3,
      completed: true,
    });
  });

  it('returns defaults when localStorage is unavailable', () => {
    vi.stubGlobal('localStorage', undefined);
    expect(loadSetupWizardState()).toEqual({ step: 0, completed: false });
  });

  it('no-ops save when localStorage is unavailable', () => {
    vi.stubGlobal('localStorage', undefined);
    expect(() => saveSetupWizardState({ step: 1, completed: false })).not.toThrow();
  });
});
