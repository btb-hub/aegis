package handler

import (
	"net/http"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TeamHandler struct {
	teams *service.TeamService
	auth  *service.AuthService
}

func NewTeamHandler(teams *service.TeamService, auth *service.AuthService) *TeamHandler {
	return &TeamHandler{teams: teams, auth: auth}
}

func (h *TeamHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))

	api.GET("/teams", h.listTeams)
	api.GET("/teams/:id", h.getTeam)
	api.GET("/teams/:id/members", h.listMembers)

	admin := api.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.POST("/teams", h.createTeam)
	admin.PATCH("/teams/:id", h.updateTeam)
	admin.DELETE("/teams/:id", h.deleteTeam)
	admin.POST("/teams/:id/members", h.addMember)
	admin.PATCH("/teams/:id/members/:userId", h.updateMember)
	admin.DELETE("/teams/:id/members/:userId", h.removeMember)
}

func (h *TeamHandler) listTeams(c *gin.Context) {
	var workspaceID *uuid.UUID
	if raw := c.Query("workspace_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			WriteError(c, apperrors.Validation("workspace_id must be a valid uuid", nil))
			return
		}
		workspaceID = &id
	}
	teams, err := h.teams.ListTeams(c.Request.Context(), workspaceID)
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

func (h *TeamHandler) createTeam(c *gin.Context) {
	var body struct {
		WorkspaceID string  `json:"workspace_id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		SupportTier *string `json:"support_tier"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	workspaceID, err := uuid.Parse(body.WorkspaceID)
	if err != nil {
		WriteError(c, apperrors.Validation("workspace_id must be a valid uuid", nil))
		return
	}
	team, err := h.teams.CreateTeam(c.Request.Context(), workspaceID, body.Name, body.Description, body.SupportTier)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusCreated, service.TeamJSON(team))
}

func (h *TeamHandler) getTeam(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	team, err := h.teams.GetTeam(c.Request.Context(), id)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.TeamJSON(team))
}

func (h *TeamHandler) updateTeam(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	var body struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		SupportTier *string `json:"support_tier"`
		WorkspaceID *string `json:"workspace_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	var workspaceID *uuid.UUID
	if body.WorkspaceID != nil && *body.WorkspaceID != "" {
		parsed, err := uuid.Parse(*body.WorkspaceID)
		if err != nil {
			WriteError(c, apperrors.Validation("workspace_id must be a valid uuid", nil))
			return
		}
		workspaceID = &parsed
	}
	team, err := h.teams.UpdateTeam(c.Request.Context(), id, body.Name, body.Description, body.SupportTier, workspaceID)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.TeamJSON(team))
}

func (h *TeamHandler) deleteTeam(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := h.teams.DeleteTeam(c.Request.Context(), id); err != nil {
		WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TeamHandler) listMembers(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	members, err := h.teams.ListMembers(c.Request.Context(), teamID)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(members))
	for _, member := range members {
		items = append(items, service.TeamMemberJSON(member))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *TeamHandler) addMember(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	var body struct {
		UserID   string `json:"user_id"`
		TeamRole string `json:"team_role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		WriteError(c, apperrors.Validation("user_id must be a valid uuid", nil))
		return
	}
	member, err := h.teams.AddMember(c.Request.Context(), teamID, userID, body.TeamRole)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusCreated, service.TeamMemberJSON(member))
}

func (h *TeamHandler) updateMember(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	userID, err := parseUUIDParam(c, "userId")
	if err != nil {
		WriteError(c, err)
		return
	}
	var body struct {
		TeamRole string `json:"team_role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	member, err := h.teams.UpdateMember(c.Request.Context(), teamID, userID, body.TeamRole)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.TeamMemberJSON(member))
}

func (h *TeamHandler) removeMember(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	userID, err := parseUUIDParam(c, "userId")
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := h.teams.RemoveMember(c.Request.Context(), teamID, userID); err != nil {
		WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		return uuid.Nil, apperrors.Validation(name+" must be a valid uuid", nil)
	}
	return id, nil
}
