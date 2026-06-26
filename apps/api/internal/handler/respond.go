package handler

import (
	"errors"
	"net/http"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
)

func WriteError(c *gin.Context, err error) {
	var appErr *apperrors.Error
	if errors.As(err, &appErr) {
		c.JSON(appErr.StatusCode, gin.H{
			"code":    appErr.Code,
			"message": appErr.Message,
			"details": appErr.Details,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"code":    "INTERNAL_ERROR",
		"message": "unexpected error",
	})
}

func WriteJSON(c *gin.Context, status int, body any) {
	c.JSON(status, body)
}
