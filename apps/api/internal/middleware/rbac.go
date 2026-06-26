package middleware

import (
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/rbac"
	"github.com/gin-gonic/gin"
)

const userRoleKey = "userRole"

func SetUserRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(userRoleKey, role)
		c.Next()
	}
}

func RequireMutate() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(userRoleKey)
		r, ok := role.(string)
		if !ok || !rbac.CanMutate(rbac.Role(r)) {
			err := apperrors.Forbidden("mutation not allowed for this role")
			c.AbortWithStatusJSON(err.StatusCode, gin.H{"code": err.Code, "message": err.Message})
			return
		}
		c.Next()
	}
}
