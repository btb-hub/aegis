package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type teamRepoMock struct {
	teams       map[uuid.UUID]db.Team
	memberships map[uuid.UUID]map[uuid.UUID]db.TeamMembership
	users       map[uuid.UUID]db.User
}

func newTeamRepoMock() *teamRepoMock {
	return &teamRepoMock{
		teams:       map[uuid.UUID]db.Team{},
		memberships: map[uuid.UUID]map[uuid.UUID]db.TeamMembership{},
		users:       map[uuid.UUID]db.User{},
	}
}

func (m *teamRepoMock) ListTeams(ctx context.Context) ([]db.Team, error) {
	items := make([]db.Team, 0, len(m.teams))
	for _, team := range m.teams {
		items = append(items, team)
	}
	return items, nil
}

func (m *teamRepoMock) GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error) {
	team, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	return team, nil
}

func (m *teamRepoMock) CreateTeam(ctx context.Context, name, description string) (db.Team, error) {
	for _, team := range m.teams {
		if team.Name == name {
			return db.Team{}, errUniqueViolation("teams_name_key")
		}
	}
	team := db.Team{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.teams[team.ID] = team
	return team, nil
}

func (m *teamRepoMock) UpdateTeam(ctx context.Context, id uuid.UUID, name, description string) (db.Team, error) {
	team, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	team.Name = name
	team.Description = description
	team.UpdatedAt = time.Now()
	m.teams[id] = team
	return team, nil
}

func (m *teamRepoMock) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.teams[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.teams, id)
	delete(m.memberships, id)
	return nil
}

func (m *teamRepoMock) ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]db.TeamMember, error) {
	byUser := m.memberships[teamID]
	items := make([]db.TeamMember, 0, len(byUser))
	for userID, membership := range byUser {
		user := m.users[userID]
		items = append(items, membershipToMember(membership, user))
	}
	return items, nil
}

func (m *teamRepoMock) AddTeamMember(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (db.TeamMembership, error) {
	if m.memberships[teamID] == nil {
		m.memberships[teamID] = map[uuid.UUID]db.TeamMembership{}
	}
	if _, exists := m.memberships[teamID][userID]; exists {
		return db.TeamMembership{}, errUniqueViolation("team_memberships_team_id_user_id_key")
	}
	membership := db.TeamMembership{
		ID:        uuid.New(),
		TeamID:    teamID,
		UserID:    userID,
		TeamRole:  teamRole,
		CreatedAt: time.Now(),
	}
	m.memberships[teamID][userID] = membership
	return membership, nil
}

func (m *teamRepoMock) UpdateTeamMemberRole(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (db.TeamMembership, error) {
	byUser, ok := m.memberships[teamID]
	if !ok {
		return db.TeamMembership{}, pgx.ErrNoRows
	}
	membership, ok := byUser[userID]
	if !ok {
		return db.TeamMembership{}, pgx.ErrNoRows
	}
	membership.TeamRole = teamRole
	byUser[userID] = membership
	return membership, nil
}

func (m *teamRepoMock) RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error {
	byUser, ok := m.memberships[teamID]
	if !ok {
		return pgx.ErrNoRows
	}
	if _, ok := byUser[userID]; !ok {
		return pgx.ErrNoRows
	}
	delete(byUser, userID)
	return nil
}

func (m *teamRepoMock) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	user, ok := m.users[id]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return user, nil
}

func errUniqueViolation(constraint string) error {
	return &pgconn.PgError{Code: "23505", ConstraintName: constraint}
}

func TestCreateTeamRequiresName(t *testing.T) {
	svc := NewTeamService(newTeamRepoMock())
	_, err := svc.CreateTeam(context.Background(), "  ", "")
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestCreateTeamSuccess(t *testing.T) {
	svc := NewTeamService(newTeamRepoMock())
	team, err := svc.CreateTeam(context.Background(), "Platform", "Core infra")
	require.NoError(t, err)
	require.Equal(t, "Platform", team.Name)
}

func TestAddMemberUnknownUser(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	team, err := svc.CreateTeam(context.Background(), "Platform", "")
	require.NoError(t, err)

	_, err = svc.AddMember(context.Background(), team.ID, uuid.New(), "member")
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestAddMemberSuccess(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	team, err := svc.CreateTeam(context.Background(), "Platform", "")
	require.NoError(t, err)
	userID := uuid.New()
	repo.users[userID] = db.User{ID: userID, Email: "u@example.com", DisplayName: "User"}

	member, err := svc.AddMember(context.Background(), team.ID, userID, "lead")
	require.NoError(t, err)
	require.Equal(t, "lead", member.TeamRole)
	require.Equal(t, "User", member.DisplayName)
}

func TestAddMemberInvalidRole(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	team, err := svc.CreateTeam(context.Background(), "Platform", "")
	require.NoError(t, err)
	userID := uuid.New()
	repo.users[userID] = db.User{ID: userID}

	_, err = svc.AddMember(context.Background(), team.ID, userID, "owner")
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestMapTeamErrorConflict(t *testing.T) {
	err := mapTeamError(errUniqueViolation("teams_name_key"))
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "CONFLICT", appErr.Code)
}

func TestMapTeamErrorNotFound(t *testing.T) {
	err := mapTeamError(pgx.ErrNoRows)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestListTeams(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	_, err := svc.CreateTeam(context.Background(), "A", "")
	require.NoError(t, err)
	_, err = svc.CreateTeam(context.Background(), "B", "")
	require.NoError(t, err)

	teams, err := svc.ListTeams(context.Background())
	require.NoError(t, err)
	require.Len(t, teams, 2)
}

func TestAddMemberDuplicateConflict(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	team, err := svc.CreateTeam(context.Background(), "Platform", "")
	require.NoError(t, err)
	userID := uuid.New()
	repo.users[userID] = db.User{ID: userID}

	_, err = svc.AddMember(context.Background(), team.ID, userID, "member")
	require.NoError(t, err)
	_, err = svc.AddMember(context.Background(), team.ID, userID, "member")
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "CONFLICT", appErr.Code)
}

func TestListMembersTeamNotFound(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	_, err := svc.ListMembers(context.Background(), uuid.New())
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestAddMemberDefaultRole(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	team, err := svc.CreateTeam(context.Background(), "Platform", "")
	require.NoError(t, err)
	userID := uuid.New()
	repo.users[userID] = db.User{ID: userID, DisplayName: "User"}

	member, err := svc.AddMember(context.Background(), team.ID, userID, "")
	require.NoError(t, err)
	require.Equal(t, "member", member.TeamRole)
}

func TestUpdateMemberNotFound(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	team, err := svc.CreateTeam(context.Background(), "Platform", "")
	require.NoError(t, err)

	_, err = svc.UpdateMember(context.Background(), team.ID, uuid.New(), "lead")
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestUpdateTeamNotFound(t *testing.T) {
	svc := NewTeamService(newTeamRepoMock())
	_, err := svc.UpdateTeam(context.Background(), uuid.New(), "Core", "")
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestUpdateMemberInvalidRole(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	team, err := svc.CreateTeam(context.Background(), "Platform", "")
	require.NoError(t, err)
	userID := uuid.New()
	repo.users[userID] = db.User{ID: userID}
	_, err = svc.AddMember(context.Background(), team.ID, userID, "member")
	require.NoError(t, err)

	_, err = svc.UpdateMember(context.Background(), team.ID, userID, "owner")
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestMapTeamErrorPassthrough(t *testing.T) {
	raw := errors.New("db down")
	require.Equal(t, raw, mapTeamError(raw))
}

func TestListMembersEmptyTeam(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	team, err := svc.CreateTeam(context.Background(), "Platform", "")
	require.NoError(t, err)

	members, err := svc.ListMembers(context.Background(), team.ID)
	require.NoError(t, err)
	require.Empty(t, members)
}

func TestUpdateAndDeleteTeam(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	team, err := svc.CreateTeam(context.Background(), "Platform", "")
	require.NoError(t, err)

	updated, err := svc.UpdateTeam(context.Background(), team.ID, "Core", "desc")
	require.NoError(t, err)
	require.Equal(t, "Core", updated.Name)

	require.NoError(t, svc.DeleteTeam(context.Background(), team.ID))
	require.Error(t, svc.DeleteTeam(context.Background(), team.ID))
}

func TestUpdateMemberAndRemoveMember(t *testing.T) {
	repo := newTeamRepoMock()
	svc := NewTeamService(repo)
	team, err := svc.CreateTeam(context.Background(), "Platform", "")
	require.NoError(t, err)
	userID := uuid.New()
	repo.users[userID] = db.User{ID: userID, DisplayName: "User"}

	_, err = svc.AddMember(context.Background(), team.ID, userID, "member")
	require.NoError(t, err)

	updated, err := svc.UpdateMember(context.Background(), team.ID, userID, "lead")
	require.NoError(t, err)
	require.Equal(t, "lead", updated.TeamRole)

	require.NoError(t, svc.RemoveMember(context.Background(), team.ID, userID))
	require.Error(t, svc.RemoveMember(context.Background(), team.ID, userID))
}

func TestGetTeamNotFound(t *testing.T) {
	svc := NewTeamService(newTeamRepoMock())
	_, err := svc.GetTeam(context.Background(), uuid.New())
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestTeamJSONHelpers(t *testing.T) {
	team := db.Team{ID: uuid.New(), Name: "Platform", Description: "desc", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	teamJSON := TeamJSON(team)
	require.Equal(t, team.ID.String(), teamJSON["id"])

	member := db.TeamMember{ID: uuid.New(), TeamID: team.ID, UserID: uuid.New(), TeamRole: "lead", Email: "u@example.com", DisplayName: "User", CreatedAt: time.Now()}
	memberJSON := TeamMemberJSON(member)
	require.Equal(t, "lead", memberJSON["team_role"])
}
