package handler

import (
	"net/http"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

type ScheduleHandler struct {
	schedules *service.ScheduleService
	auth      *service.AuthService
}

func NewScheduleHandler(schedules *service.ScheduleService, auth *service.AuthService) *ScheduleHandler {
	return &ScheduleHandler{schedules: schedules, auth: auth}
}

func (h *ScheduleHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))

	api.GET("/teams/:id/schedules", h.listSchedules)
	api.GET("/teams/:id/schedules/:sid", h.getSchedule)

	admin := api.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.POST("/teams/:id/schedules", h.createSchedule)
	admin.PATCH("/teams/:id/schedules/:sid", h.updateSchedule)
	admin.DELETE("/teams/:id/schedules/:sid", h.deleteSchedule)
}

func (h *ScheduleHandler) listSchedules(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	schedules, err := h.schedules.ListSchedules(c.Request.Context(), teamID)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(schedules))
	for _, schedule := range schedules {
		items = append(items, service.ScheduleJSON(schedule))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *ScheduleHandler) getSchedule(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	scheduleID, err := parseUUIDParam(c, "sid")
	if err != nil {
		WriteError(c, err)
		return
	}
	schedule, err := h.schedules.GetSchedule(c.Request.Context(), teamID, scheduleID)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.ScheduleJSON(schedule))
}

func (h *ScheduleHandler) createSchedule(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	input, err := bindScheduleInput(c)
	if err != nil {
		WriteError(c, err)
		return
	}
	schedule, err := h.schedules.CreateSchedule(c.Request.Context(), teamID, input)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusCreated, service.ScheduleJSON(schedule))
}

func (h *ScheduleHandler) updateSchedule(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	scheduleID, err := parseUUIDParam(c, "sid")
	if err != nil {
		WriteError(c, err)
		return
	}
	input, err := bindScheduleInput(c)
	if err != nil {
		WriteError(c, err)
		return
	}
	schedule, err := h.schedules.UpdateSchedule(c.Request.Context(), teamID, scheduleID, input)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.ScheduleJSON(schedule))
}

func (h *ScheduleHandler) deleteSchedule(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	scheduleID, err := parseUUIDParam(c, "sid")
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := h.schedules.DeleteSchedule(c.Request.Context(), teamID, scheduleID); err != nil {
		WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func bindScheduleInput(c *gin.Context) (service.CreateScheduleInput, error) {
	var body struct {
		Name     string `json:"name"`
		Timezone string `json:"timezone"`
		Rotation struct {
			HandoffWeekday int32    `json:"handoff_weekday"`
			HandoffTime    string   `json:"handoff_time"`
			Participants   []string `json:"participants"`
		} `json:"rotation"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return service.CreateScheduleInput{}, service.ErrInvalidBody()
	}
	participants, err := service.ParseParticipantIDs(body.Rotation.Participants)
	if err != nil {
		return service.CreateScheduleInput{}, err
	}
	return service.CreateScheduleInput{
		Name:     body.Name,
		Timezone: body.Timezone,
		Rotation: service.WeeklyRotationInput{
			HandoffWeekday: body.Rotation.HandoffWeekday,
			HandoffTime:    body.Rotation.HandoffTime,
			Participants:   participants,
		},
	}, nil
}
