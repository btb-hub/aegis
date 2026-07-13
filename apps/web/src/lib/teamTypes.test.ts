import { describe, expect, it } from 'vitest';
import {
  bounceLabelKey,
  handoffLabelKey,
  handoffTeamLabelKey,
  validEscalationTargetTiers,
} from './teamTypes';

describe('teamTypes helpers', () => {
  it('returns valid escalation targets per tier', () => {
    expect(validEscalationTargetTiers('noc')).toEqual(['l1', 'l2']);
    expect(validEscalationTargetTiers('l1')).toEqual(['l2']);
    expect(validEscalationTargetTiers('l2')).toEqual(['l3']);
    expect(validEscalationTargetTiers('l3')).toEqual([]);
  });

  it('maps handoff labels from owning tier', () => {
    expect(handoffLabelKey('noc')).toBe('incidents.escalate_to_l1');
    expect(handoffLabelKey('l1')).toBe('incidents.escalate_to_l2');
    expect(handoffLabelKey('l2')).toBe('incidents.handoff_to_l3');
    expect(handoffLabelKey()).toBe('incidents.handoff');
  });

  it('maps handoff team field labels from owning tier', () => {
    expect(handoffTeamLabelKey('noc')).toBe('incidents.handoff_team_label_l1');
    expect(handoffTeamLabelKey('l1')).toBe('incidents.handoff_team_label_l2');
    expect(handoffTeamLabelKey('l2')).toBe('incidents.handoff_team_label_l3');
    expect(handoffTeamLabelKey()).toBe('incidents.handoff_team_label');
  });

  it('maps bounce labels from owning tier', () => {
    expect(bounceLabelKey('l1')).toBe('incidents.bounce_to_l1');
    expect(bounceLabelKey('l2')).toBe('incidents.bounce_to_l2');
    expect(bounceLabelKey('l3')).toBe('incidents.bounce_to_l2');
    expect(bounceLabelKey()).toBe('incidents.bounce');
  });
});
