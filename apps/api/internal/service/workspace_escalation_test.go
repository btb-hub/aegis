package service

import (
	"context"
	"errors"
	"testing"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type workspaceRepoMock struct {
	items               map[uuid.UUID]db.Workspace
	slotsProvisionedFor []uuid.UUID
}

func (m *workspaceRepoMock) ListWorkspaces(ctx context.Context) ([]db.Workspace, error) {
	items := make([]db.Workspace, 0, len(m.items))
	for _, item := range m.items {
		items = append(items, item)
	}
	return items, nil
}

func (m *workspaceRepoMock) GetWorkspace(_ context.Context, id uuid.UUID) (db.Workspace, error) {
	item, ok := m.items[id]
	if !ok {
		return db.Workspace{}, pgx.ErrNoRows
	}
	return item, nil
}

func (m *workspaceRepoMock) CreateWorkspace(_ context.Context, name, slug, description string) (db.Workspace, error) {
	item := db.Workspace{ID: uuid.New(), Name: name, Slug: slug, Description: description}
	m.items[item.ID] = item
	return item, nil
}

func (m *workspaceRepoMock) EnsureWorkspaceSlots(_ context.Context, workspaceID uuid.UUID) error {
	m.slotsProvisionedFor = append(m.slotsProvisionedFor, workspaceID)
	return nil
}

func (m *workspaceRepoMock) UpdateWorkspace(_ context.Context, id uuid.UUID, name, slug, description string) (db.Workspace, error) {
	item, ok := m.items[id]
	if !ok {
		return db.Workspace{}, pgx.ErrNoRows
	}
	item.Name = name
	item.Slug = slug
	item.Description = description
	m.items[id] = item
	return item, nil
}

func (m *workspaceRepoMock) ListWorkspacesWithCounts(_ context.Context) ([]db.WorkspaceSummary, error) {
	items := make([]db.WorkspaceSummary, 0, len(m.items))
	for _, item := range m.items {
		items = append(items, db.WorkspaceSummary{Workspace: item})
	}
	return items, nil
}

func (m *workspaceRepoMock) GetWorkspaceUsage(_ context.Context, id uuid.UUID) (db.WorkspaceUsage, error) {
	if _, ok := m.items[id]; !ok {
		return db.WorkspaceUsage{}, pgx.ErrNoRows
	}
	return db.WorkspaceUsage{}, nil
}

func (m *workspaceRepoMock) DeleteWorkspace(_ context.Context, id uuid.UUID) error {
	if _, ok := m.items[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.items, id)
	return nil
}

func TestWorkspaceServiceCreate(t *testing.T) {
	repo := &workspaceRepoMock{items: map[uuid.UUID]db.Workspace{}}
	svc := NewWorkspaceService(repo)
	item, err := svc.Create(context.Background(), "Platform", "platform", "Core")
	require.NoError(t, err)
	require.Equal(t, "platform", item.Slug)
	require.Equal(t, []uuid.UUID{item.ID}, repo.slotsProvisionedFor)
}

func TestWorkspaceServiceCreateRequiresName(t *testing.T) {
	svc := NewWorkspaceService(&workspaceRepoMock{items: map[uuid.UUID]db.Workspace{}})
	_, err := svc.Create(context.Background(), "  ", "", "")
	require.Error(t, err)
}

func TestWorkspaceServiceGetNotFound(t *testing.T) {
	svc := NewWorkspaceService(&workspaceRepoMock{items: map[uuid.UUID]db.Workspace{}})
	_, err := svc.Get(context.Background(), uuid.New())
	require.Error(t, err)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

type escalationRepoMock struct {
	teams map[uuid.UUID]db.Team
	paths []db.EscalationPath
}

func (m *escalationRepoMock) GetTeam(_ context.Context, id uuid.UUID) (db.Team, error) {
	team, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	return team, nil
}

func (m *escalationRepoMock) ListEscalationPathsByWorkspace(context.Context, uuid.UUID) ([]db.EscalationPath, error) {
	return m.paths, nil
}

func (m *escalationRepoMock) ListEscalationPathsFromTeam(_ context.Context, fromTeamID uuid.UUID) ([]db.EscalationPath, error) {
	var out []db.EscalationPath
	for _, path := range m.paths {
		if path.FromTeamID == fromTeamID {
			out = append(out, path)
		}
	}
	return out, nil
}

func (m *escalationRepoMock) ListEscalationPathsToTeam(_ context.Context, toTeamID uuid.UUID) ([]db.EscalationPath, error) {
	var out []db.EscalationPath
	for _, path := range m.paths {
		if path.ToTeamID == toTeamID {
			out = append(out, path)
		}
	}
	return out, nil
}

func (m *escalationRepoMock) ReplaceEscalationPaths(_ context.Context, workspaceID uuid.UUID, paths []db.EscalationPath) error {
	m.paths = paths
	return nil
}

func (m *escalationRepoMock) AddEscalationPath(_ context.Context, path db.EscalationPath) (db.EscalationPath, error) {
	path.ID = uuid.New()
	m.paths = append(m.paths, path)
	return path, nil
}

func (m *escalationRepoMock) DeleteEscalationPath(_ context.Context, id uuid.UUID) error {
	for i, path := range m.paths {
		if path.ID == id {
			m.paths = append(m.paths[:i], m.paths[i+1:]...)
			return nil
		}
	}
	return pgx.ErrNoRows
}

func (m *escalationRepoMock) HasEscalationPath(_ context.Context, fromTeamID, toTeamID uuid.UUID) (bool, error) {
	for _, path := range m.paths {
		if path.FromTeamID == fromTeamID && path.ToTeamID == toTeamID {
			return true, nil
		}
	}
	return false, nil
}

func (m *escalationRepoMock) ListHandoffTargetTeams(_ context.Context, fromTeamID uuid.UUID) ([]db.Team, error) {
	var out []db.Team
	for _, path := range m.paths {
		if path.FromTeamID == fromTeamID {
			if team, ok := m.teams[path.ToTeamID]; ok {
				out = append(out, team)
			}
		}
	}
	return out, nil
}

func tierPtr(tier string) *string { return &tier }

func TestEscalationValidateTierAdjacency(t *testing.T) {
	workspaceID := uuid.New()
	l2ID := uuid.New()
	l3ID := uuid.New()
	repo := &escalationRepoMock{
		teams: map[uuid.UUID]db.Team{
			l2ID: {ID: l2ID, WorkspaceID: workspaceID, SupportTier: tierPtr("l2")},
			l3ID: {ID: l3ID, WorkspaceID: workspaceID, SupportTier: tierPtr("l3")},
		},
	}
	svc := NewEscalationService(repo)
	path, err := svc.AddPath(context.Background(), workspaceID, EscalationPathInput{
		FromTeamID: l2ID,
		ToTeamID:   l3ID,
	})
	require.NoError(t, err)
	require.Equal(t, l2ID, path.FromTeamID)
}

func TestEscalationRejectsInvalidTierPair(t *testing.T) {
	workspaceID := uuid.New()
	l2a := uuid.New()
	l2b := uuid.New()
	repo := &escalationRepoMock{
		teams: map[uuid.UUID]db.Team{
			l2a: {ID: l2a, WorkspaceID: workspaceID, SupportTier: tierPtr("l2")},
			l2b: {ID: l2b, WorkspaceID: workspaceID, SupportTier: tierPtr("l2")},
		},
	}
	svc := NewEscalationService(repo)
	_, err := svc.AddPath(context.Background(), workspaceID, EscalationPathInput{
		FromTeamID: l2a,
		ToTeamID:   l2b,
	})
	require.Error(t, err)
}

func TestHandoffRejectsMissingPath(t *testing.T) {
	incidentID := uuid.New()
	fromTeamID := uuid.New()
	toTeamID := uuid.New()
	repo := &handoffNoPathRepo{
		handoffMockRepo: handoffMockRepo{
			incident: db.Incident{ID: incidentID, TeamID: fromTeamID},
			team:     db.Team{ID: toTeamID},
			onCall:   []db.OnCallUser{{UserID: uuid.New()}},
		},
	}
	svc := NewHandoffService(repo)
	_, err := svc.Handoff(context.Background(), incidentID, uuid.New(), toTeamID, "note")
	require.Error(t, err)
}

type handoffNoPathRepo struct {
	handoffMockRepo
}

func (m *handoffNoPathRepo) HasEscalationPath(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

func TestNormalizeSupportTier(t *testing.T) {
	tier := "l2"
	out, err := normalizeSupportTier(&tier)
	require.NoError(t, err)
	require.Equal(t, "l2", *out)

	bad := "invalid"
	_, err = normalizeSupportTier(&bad)
	require.Error(t, err)
}

func TestWorkspaceServiceListUpdateDeleteJSON(t *testing.T) {
	id := uuid.New()
	repo := &workspaceRepoMock{items: map[uuid.UUID]db.Workspace{
		id: {ID: id, Name: "Platform", Slug: "platform", Description: "Core"},
	}}
	svc := NewWorkspaceService(repo)

	items, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)

	item, err := svc.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "Platform", item.Name)

	updated, err := svc.Update(context.Background(), id, "Platform Ops", "", "Updated")
	require.NoError(t, err)
	require.Equal(t, "platform-ops", updated.Slug)

	json := WorkspaceJSON(updated)
	require.Equal(t, id.String(), json["id"])

	require.NoError(t, svc.Delete(context.Background(), id))
	_, err = svc.Get(context.Background(), id)
	require.Error(t, err)
}

func TestWorkspaceServiceUpdateRequiresName(t *testing.T) {
	svc := NewWorkspaceService(&workspaceRepoMock{items: map[uuid.UUID]db.Workspace{}})
	_, err := svc.Update(context.Background(), uuid.New(), "  ", "", "")
	require.Error(t, err)
}

func TestEscalationServiceCoverage(t *testing.T) {
	workspaceID := uuid.New()
	l2ID := uuid.New()
	l3ID := uuid.New()
	repo := &escalationRepoMock{
		teams: map[uuid.UUID]db.Team{
			l2ID: {ID: l2ID, WorkspaceID: workspaceID, SupportTier: tierPtr("l2")},
			l3ID: {ID: l3ID, WorkspaceID: workspaceID, SupportTier: tierPtr("l3")},
		},
	}
	svc := NewEscalationService(repo)

	paths, err := svc.ListByWorkspace(context.Background(), workspaceID)
	require.NoError(t, err)
	require.Empty(t, paths)

	_, err = svc.AddPath(context.Background(), workspaceID, EscalationPathInput{
		FromTeamID: l2ID,
		ToTeamID:   l3ID,
	})
	require.NoError(t, err)

	fromPaths, err := svc.ListFromTeam(context.Background(), l2ID)
	require.NoError(t, err)
	require.Len(t, fromPaths, 1)

	toPaths, err := svc.ListToTeam(context.Background(), l3ID)
	require.NoError(t, err)
	require.Len(t, toPaths, 1)

	targets, err := svc.HandoffTargets(context.Background(), l2ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)

	replaced, err := svc.ReplaceWorkspacePaths(context.Background(), workspaceID, []EscalationPathInput{{
		FromTeamID: l2ID,
		ToTeamID:   l3ID,
	}})
	require.NoError(t, err)
	require.Len(t, replaced, 1)

	require.NoError(t, svc.ValidateHandoffTarget(context.Background(), l2ID, l3ID))
	require.NoError(t, svc.DeletePath(context.Background(), replaced[0].ID))

	_, err = svc.HandoffTargets(context.Background(), uuid.New())
	require.Error(t, err)

	json := EscalationPathJSON(replaced[0])
	require.Equal(t, l2ID.String(), json["from_team_id"])
}

func TestEscalationRejectsSameTeam(t *testing.T) {
	workspaceID := uuid.New()
	teamID := uuid.New()
	repo := &escalationRepoMock{
		teams: map[uuid.UUID]db.Team{
			teamID: {ID: teamID, WorkspaceID: workspaceID, SupportTier: tierPtr("l2")},
		},
	}
	svc := NewEscalationService(repo)
	_, err := svc.AddPath(context.Background(), workspaceID, EscalationPathInput{
		FromTeamID: teamID,
		ToTeamID:   teamID,
	})
	require.Error(t, err)
}

func TestEscalationDeletePathNotFound(t *testing.T) {
	svc := NewEscalationService(&escalationRepoMock{})
	err := svc.DeletePath(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestEscalationValidateHandoffTargetMissingPath(t *testing.T) {
	svc := NewEscalationService(&escalationRepoMock{})
	err := svc.ValidateHandoffTarget(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
}

func TestMapWorkspaceErrorConflict(t *testing.T) {
	err := mapWorkspaceError(&pgconn.PgError{Code: "23505"})
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "CONFLICT", appErr.Code)
}

func TestMapWorkspaceErrorPassthrough(t *testing.T) {
	inner := errors.New("db down")
	require.Equal(t, inner, mapWorkspaceError(inner))
}

func TestWorkspaceServiceDeleteNotFound(t *testing.T) {
	svc := NewWorkspaceService(&workspaceRepoMock{items: map[uuid.UUID]db.Workspace{}})
	err := svc.Delete(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestEscalationValidatePathCrossWorkspace(t *testing.T) {
	workspaceA := uuid.New()
	workspaceB := uuid.New()
	l2ID := uuid.New()
	l3ID := uuid.New()
	repo := &escalationRepoMock{
		teams: map[uuid.UUID]db.Team{
			l2ID: {ID: l2ID, WorkspaceID: workspaceA, SupportTier: tierPtr("l2")},
			l3ID: {ID: l3ID, WorkspaceID: workspaceB, SupportTier: tierPtr("l3")},
		},
	}
	svc := NewEscalationService(repo)
	_, err := svc.AddPath(context.Background(), workspaceA, EscalationPathInput{
		FromTeamID: l2ID,
		ToTeamID:   l3ID,
	})
	require.Error(t, err)

	_, err = svc.AddPath(context.Background(), workspaceA, EscalationPathInput{
		FromTeamID:     l2ID,
		ToTeamID:       l3ID,
		CrossWorkspace: true,
	})
	require.NoError(t, err)
}

func TestEscalationMapperPassthrough(t *testing.T) {
	inner := errors.New("db down")
	require.Equal(t, inner, mapEscalationTeamError(inner))
	require.Equal(t, inner, mapEscalationPathError(inner))
}

func TestEscalationValidatePathMissingTier(t *testing.T) {
	workspaceID := uuid.New()
	fromID := uuid.New()
	toID := uuid.New()
	repo := &escalationRepoMock{
		teams: map[uuid.UUID]db.Team{
			fromID: {ID: fromID, WorkspaceID: workspaceID, SupportTier: tierPtr("l2")},
			toID:   {ID: toID, WorkspaceID: workspaceID},
		},
	}
	svc := NewEscalationService(repo)
	_, err := svc.AddPath(context.Background(), workspaceID, EscalationPathInput{
		FromTeamID: fromID,
		ToTeamID:   toID,
	})
	require.Error(t, err)
}

func TestEscalationValidatePathTeamNotInWorkspace(t *testing.T) {
	workspaceA := uuid.New()
	workspaceB := uuid.New()
	otherWorkspace := uuid.New()
	fromID := uuid.New()
	toID := uuid.New()
	repo := &escalationRepoMock{
		teams: map[uuid.UUID]db.Team{
			fromID: {ID: fromID, WorkspaceID: workspaceA, SupportTier: tierPtr("l2")},
			toID:   {ID: toID, WorkspaceID: workspaceB, SupportTier: tierPtr("l3")},
		},
	}
	svc := NewEscalationService(repo)
	_, err := svc.AddPath(context.Background(), otherWorkspace, EscalationPathInput{
		FromTeamID: fromID,
		ToTeamID:   toID,
	})
	require.Error(t, err)
}

func TestEscalationValidateTierAdjacencyInvalidSource(t *testing.T) {
	err := validateTierAdjacency(tierPtr("l3"), tierPtr("l2"))
	require.Error(t, err)
}

func TestEscalationValidateTierAdjacencyMissingTier(t *testing.T) {
	err := validateTierAdjacency(nil, tierPtr("l2"))
	require.Error(t, err)
}

func TestEscalationListByWorkspaceNilPaths(t *testing.T) {
	repo := &escalationRepoMockNilPaths{}
	svc := NewEscalationService(repo)
	paths, err := svc.ListByWorkspace(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Empty(t, paths)
}

type escalationRepoMockNilPaths struct {
	escalationRepoMock
}

func (m *escalationRepoMockNilPaths) ListEscalationPathsByWorkspace(context.Context, uuid.UUID) ([]db.EscalationPath, error) {
	return nil, nil
}

func TestWorkspaceListWithCounts(t *testing.T) {
	wsID := uuid.New()
	repo := &workspaceRepoMockWithCounts{
		workspaceRepoMock: workspaceRepoMock{items: map[uuid.UUID]db.Workspace{wsID: {ID: wsID, Name: "Platform"}}},
		summaries: []db.WorkspaceSummary{{
			Workspace: db.Workspace{ID: wsID, Name: "Platform", Slug: "platform"},
			TeamCount: 2, RoutingRuleCount: 1,
		}},
	}
	svc := NewWorkspaceService(repo)
	items, err := svc.ListWithCounts(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, 2, items[0].TeamCount)
}

type workspaceRepoMockWithCounts struct {
	workspaceRepoMock
	summaries []db.WorkspaceSummary
}

func (m *workspaceRepoMockWithCounts) ListWorkspacesWithCounts(context.Context) ([]db.WorkspaceSummary, error) {
	return m.summaries, nil
}

func TestWorkspaceDeleteDefaultForbidden(t *testing.T) {
	repo := &workspaceRepoMock{items: map[uuid.UUID]db.Workspace{
		db.DefaultWorkspaceID: {ID: db.DefaultWorkspaceID, Name: "Default", Slug: "default"},
	}}
	svc := NewWorkspaceService(repo)
	err := svc.Delete(context.Background(), db.DefaultWorkspaceID)
	require.Error(t, err)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "FORBIDDEN", appErr.Code)
}

func TestWorkspaceDeleteNotEmpty(t *testing.T) {
	wsID := uuid.New()
	repo := &workspaceRepoMockWithUsage{
		workspaceRepoMock: workspaceRepoMock{items: map[uuid.UUID]db.Workspace{wsID: {ID: wsID, Name: "Platform"}}},
		usage:             db.WorkspaceUsage{TeamCount: 1},
	}
	svc := NewWorkspaceService(repo)
	err := svc.Delete(context.Background(), wsID)
	require.Error(t, err)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "CONFLICT", appErr.Code)
}

func TestWorkspaceDeleteSuccess(t *testing.T) {
	wsID := uuid.New()
	repo := &workspaceRepoMock{items: map[uuid.UUID]db.Workspace{wsID: {ID: wsID, Name: "Platform", Slug: "platform"}}}
	svc := NewWorkspaceService(repo)
	err := svc.Delete(context.Background(), wsID)
	require.NoError(t, err)
	_, err = svc.Get(context.Background(), wsID)
	require.Error(t, err)
}

type workspaceRepoMockWithUsage struct {
	workspaceRepoMock
	usage db.WorkspaceUsage
}

func (m *workspaceRepoMockWithUsage) GetWorkspaceUsage(context.Context, uuid.UUID) (db.WorkspaceUsage, error) {
	return m.usage, nil
}

func TestWorkspaceSummaryJSON(t *testing.T) {
	wsID := uuid.New()
	out := WorkspaceSummaryJSON(db.WorkspaceSummary{
		Workspace:        db.Workspace{ID: wsID, Name: "Platform", Slug: "platform"},
		TeamCount:        3,
		RoutingRuleCount: 2,
	})
	require.Equal(t, 3, out["team_count"])
	require.Equal(t, 2, out["routing_rule_count"])
}

func TestBlockedTeamsForWorkspaceMoveCoMove(t *testing.T) {
	defaultWS := db.DefaultWorkspaceID
	targetWS := uuid.New()
	l2ID := uuid.New()
	l3ID := uuid.New()
	repo := &escalationRepoMock{
		teams: map[uuid.UUID]db.Team{
			l2ID: {ID: l2ID, WorkspaceID: defaultWS, SupportTier: tierPtr("l2")},
			l3ID: {ID: l3ID, WorkspaceID: defaultWS, SupportTier: tierPtr("l3")},
		},
		paths: []db.EscalationPath{{
			ID: uuid.New(), FromTeamID: l2ID, ToTeamID: l3ID, CrossWorkspace: false,
		}},
	}
	svc := NewEscalationService(repo)
	blocked, err := svc.BlockedTeamsForWorkspaceMove(context.Background(), targetWS, []uuid.UUID{l2ID, l3ID})
	require.NoError(t, err)
	require.Empty(t, blocked)
}

func TestBlockedTeamsForWorkspaceMoveSingleBlocked(t *testing.T) {
	defaultWS := db.DefaultWorkspaceID
	targetWS := uuid.New()
	otherWS := uuid.New()
	l2ID := uuid.New()
	l3ID := uuid.New()
	repo := &escalationRepoMock{
		teams: map[uuid.UUID]db.Team{
			l2ID: {ID: l2ID, WorkspaceID: defaultWS, SupportTier: tierPtr("l2")},
			l3ID: {ID: l3ID, WorkspaceID: otherWS, SupportTier: tierPtr("l3")},
		},
		paths: []db.EscalationPath{{
			ID: uuid.New(), FromTeamID: l2ID, ToTeamID: l3ID, CrossWorkspace: false,
		}},
	}
	svc := NewEscalationService(repo)
	blocked, err := svc.BlockedTeamsForWorkspaceMove(context.Background(), targetWS, []uuid.UUID{l2ID})
	require.NoError(t, err)
	require.NotEmpty(t, blocked[l2ID])
}

func TestBlockedTeamsForWorkspaceMoveAlreadyInTarget(t *testing.T) {
	targetWS := uuid.New()
	l2ID := uuid.New()
	repo := &escalationRepoMock{
		teams: map[uuid.UUID]db.Team{
			l2ID: {ID: l2ID, WorkspaceID: targetWS, SupportTier: tierPtr("l2")},
		},
	}
	svc := NewEscalationService(repo)
	_, err := svc.BlockedTeamsForWorkspaceMove(context.Background(), targetWS, []uuid.UUID{l2ID})
	require.Error(t, err)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestBlockedTeamsJSON(t *testing.T) {
	pathID := uuid.New()
	l2ID := uuid.New()
	items := BlockedTeamsJSON(map[uuid.UUID][]db.EscalationPath{
		l2ID: {{ID: pathID, FromTeamID: l2ID, ToTeamID: uuid.New()}},
	})
	require.Len(t, items, 1)
	require.Equal(t, l2ID.String(), items[0]["team_id"])
}
