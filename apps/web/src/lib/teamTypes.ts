export type SupportTier = 'l1' | 'l2' | 'l3' | 'noc';

export const DEFAULT_WORKSPACE_ID = '00000000-0000-0000-0000-000000000001';

export const SUPPORT_TIERS: SupportTier[] = ['noc', 'l1', 'l2', 'l3'];

export function validEscalationTargetTiers(fromTier: SupportTier): SupportTier[] {
  switch (fromTier) {
    case 'noc':
      return ['l1', 'l2'];
    case 'l1':
      return ['l2'];
    case 'l2':
      return ['l3'];
    default:
      return [];
  }
}

export function handoffLabelKey(tier?: string): string {
  switch (tier) {
    case 'noc':
      return 'incidents.escalate_to_l1';
    case 'l1':
      return 'incidents.escalate_to_l2';
    case 'l2':
      return 'incidents.handoff_to_l3';
    default:
      return 'incidents.handoff';
  }
}

export function bounceLabelKey(tier?: string): string {
  switch (tier) {
    case 'l1':
      return 'incidents.bounce_to_l1';
    case 'l2':
    case 'l3':
      return 'incidents.bounce_to_l2';
    default:
      return 'incidents.bounce';
  }
}

export type Workspace = {
  id: string;
  name: string;
  slug: string;
  description: string;
  created_at: string;
  updated_at: string;
};

export type Team = {
  id: string;
  workspace_id: string;
  name: string;
  description: string;
  support_tier?: SupportTier;
  created_at: string;
  updated_at: string;
};

export type TeamMember = {
  id: string;
  team_id: string;
  user_id: string;
  team_role: 'member' | 'lead';
  email: string;
  display_name: string;
  created_at: string;
};

export type UserDirectoryItem = {
  id: string;
  email: string;
  display_name: string;
  role: string;
  avatar_url?: string | null;
};

export type TeamRole = TeamMember['team_role'];

export const TEAM_ROLES: TeamRole[] = ['member', 'lead'];
