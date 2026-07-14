package handler

import (
	"net/http"
	"strings"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WorkspaceHandler struct {
	workspaces *service.WorkspaceService
	teams      *service.TeamService
	auth       *service.AuthService
}

func NewWorkspaceHandler(workspaces *service.WorkspaceService, teams *service.TeamService, auth *service.AuthService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaces: workspaces, teams: teams, auth: auth}
}

func (h *WorkspaceHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))

	api.GET("/workspaces", h.listWorkspaces)
	api.GET("/workspaces/:id", h.getWorkspace)

	admin := api.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.POST("/workspaces", h.createWorkspace)
	admin.PATCH("/workspaces/:id", h.updateWorkspace)
	admin.DELETE("/workspaces/:id", h.deleteWorkspace)
	admin.POST("/workspaces/:id/teams", h.assignTeams)
}

func (h *WorkspaceHandler) listWorkspaces(c *gin.Context) {
	items, err := h.workspaces.ListWithCounts(c.Request.Context())
	if err != nil {
		WriteError(c, err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, service.WorkspaceSummaryJSON(item))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": out})
}

func (h *WorkspaceHandler) getWorkspace(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	item, err := h.workspaces.Get(c.Request.Context(), id)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.WorkspaceJSON(item))
}

func (h *WorkspaceHandler) createWorkspace(c *gin.Context) {
	var body struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	item, err := h.workspaces.Create(c.Request.Context(), body.Name, body.Slug, body.Description)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusCreated, service.WorkspaceJSON(item))
}

func (h *WorkspaceHandler) updateWorkspace(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	var body struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	item, err := h.workspaces.Update(c.Request.Context(), id, body.Name, body.Slug, body.Description)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.WorkspaceJSON(item))
}

func (h *WorkspaceHandler) deleteWorkspace(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := h.workspaces.Delete(c.Request.Context(), id); err != nil {
		WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *WorkspaceHandler) assignTeams(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	var body struct {
		TeamIDs []string `json:"team_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	if len(body.TeamIDs) == 0 {
		WriteError(c, apperrors.Validation("team_ids must not be empty", nil))
		return
	}
	teamIDs := make([]uuid.UUID, 0, len(body.TeamIDs))
	for _, raw := range body.TeamIDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			WriteError(c, apperrors.Validation("team_ids must contain valid UUIDs", nil))
			return
		}
		teamIDs = append(teamIDs, id)
	}
	teams, err := h.teams.MoveTeamsToWorkspace(c.Request.Context(), workspaceID, teamIDs)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(teams))
	for _, team := range teams {
		items = append(items, service.TeamJSON(team))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

type EscalationHandler struct {
	escalation *service.EscalationService
	auth       *service.AuthService
}

func NewEscalationHandler(escalation *service.EscalationService, auth *service.AuthService) *EscalationHandler {
	return &EscalationHandler{escalation: escalation, auth: auth}
}

func (h *EscalationHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))

	api.GET("/teams/:id/handoff-targets", h.handoffTargets)
	api.GET("/teams/:id/escalation-paths/outgoing", h.outgoingPaths)
	api.GET("/teams/:id/escalation-paths/incoming", h.incomingPaths)
	api.GET("/workspaces/:id/escalation-paths", h.listWorkspacePaths)

	admin := api.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.PUT("/workspaces/:id/escalation-paths", h.replaceWorkspacePaths)
	admin.POST("/workspaces/:id/escalation-paths", h.addPath)
	admin.DELETE("/escalation-paths/:id", h.deletePath)
}

func (h *EscalationHandler) handoffTargets(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	teams, err := h.escalation.HandoffTargets(c.Request.Context(), id)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(teams))
	for _, team := range teams {
		items = append(items, service.TeamJSON(team))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *EscalationHandler) outgoingPaths(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	paths, err := h.escalation.ListFromTeam(c.Request.Context(), id)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		items = append(items, service.EscalationPathJSON(path))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *EscalationHandler) incomingPaths(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	paths, err := h.escalation.ListToTeam(c.Request.Context(), id)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		items = append(items, service.EscalationPathJSON(path))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *EscalationHandler) listWorkspacePaths(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	paths, err := h.escalation.ListByWorkspace(c.Request.Context(), id)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		items = append(items, service.EscalationPathJSON(path))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *EscalationHandler) replaceWorkspacePaths(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	var body struct {
		Paths []struct {
			FromTeamID     string `json:"from_team_id"`
			ToTeamID       string `json:"to_team_id"`
			CrossWorkspace bool   `json:"cross_workspace"`
		} `json:"paths"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	inputs := make([]service.EscalationPathInput, 0, len(body.Paths))
	for _, path := range body.Paths {
		fromID, err := uuid.Parse(path.FromTeamID)
		if err != nil {
			WriteError(c, service.ErrInvalidBody())
			return
		}
		toID, err := uuid.Parse(path.ToTeamID)
		if err != nil {
			WriteError(c, service.ErrInvalidBody())
			return
		}
		inputs = append(inputs, service.EscalationPathInput{
			FromTeamID:     fromID,
			ToTeamID:       toID,
			CrossWorkspace: path.CrossWorkspace,
		})
	}
	paths, err := h.escalation.ReplaceWorkspacePaths(c.Request.Context(), id, inputs)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		items = append(items, service.EscalationPathJSON(path))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *EscalationHandler) addPath(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	var body struct {
		FromTeamID     string `json:"from_team_id"`
		ToTeamID       string `json:"to_team_id"`
		CrossWorkspace bool   `json:"cross_workspace"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	fromID, err := uuid.Parse(body.FromTeamID)
	if err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	toID, err := uuid.Parse(body.ToTeamID)
	if err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	path, err := h.escalation.AddPath(c.Request.Context(), workspaceID, service.EscalationPathInput{
		FromTeamID:     fromID,
		ToTeamID:       toID,
		CrossWorkspace: body.CrossWorkspace,
	})
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusCreated, service.EscalationPathJSON(path))
}

func (h *EscalationHandler) deletePath(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := h.escalation.DeletePath(c.Request.Context(), id); err != nil {
		WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
