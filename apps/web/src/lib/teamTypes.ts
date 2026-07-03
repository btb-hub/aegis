export type Team = {
  id: string;
  name: string;
  description: string;
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
