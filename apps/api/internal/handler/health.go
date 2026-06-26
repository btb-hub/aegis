package handler

import (
	"net/http"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	health *service.HealthService
}

func NewHealthHandler(health *service.HealthService) *HealthHandler {
	return &HealthHandler{health: health}
}

func (h *HealthHandler) Register(r gin.IRouter) {
	r.GET("/healthz", h.healthz)
	r.GET("/readyz", h.readyz)
}

func (h *HealthHandler) healthz(c *gin.Context) {
	if h.health.Live() {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down"})
}

func (h *HealthHandler) readyz(c *gin.Context) {
	if err := h.health.Ready(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
