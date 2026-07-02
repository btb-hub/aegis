const STORAGE_KEY = 'aegis_setup_wizard';

export type SetupWizardState = {
  step: number;
  completed: boolean;
};

export function loadSetupWizardState(): SetupWizardState {
  if (typeof localStorage === 'undefined') {
    return { step: 0, completed: false };
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return { step: 0, completed: false };
    }
    const parsed = JSON.parse(raw) as SetupWizardState;
    if (typeof parsed.step !== 'number') {
      return { step: 0, completed: false };
    }
    return {
      step: Math.max(0, Math.min(parsed.step, 4)),
      completed: Boolean(parsed.completed),
    };
  } catch {
    return { step: 0, completed: false };
  }
}

export function saveSetupWizardState(state: SetupWizardState): void {
  if (typeof localStorage === 'undefined') {
    return;
  }
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
}
