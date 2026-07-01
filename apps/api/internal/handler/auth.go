package handler

import (
	"net/http"
	"strings"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

const sessionCookie = "aegis_session"

type AuthHandler struct {
	auth      *service.AuthService
	publicURL string
}

func NewAuthHandler(auth *service.AuthService, publicURL string) *AuthHandler {
	return &AuthHandler{auth: auth, publicURL: publicURL}
}

func (h *AuthHandler) Register(r gin.IRouter) {
	r.GET("/auth/:provider/login", h.login)
	r.GET("/auth/:provider/callback", h.callback)
	r.POST("/auth/logout", h.logout)
	r.GET("/auth/me", h.me)
	r.PATCH("/auth/me", h.patchMe)
}

func (h *AuthHandler) login(c *gin.Context) {
	provider := c.Param("provider")
	url, state, err := h.auth.LoginURL(provider)
	if err != nil {
		WriteError(c, err)
		return
	}
	c.SetCookie("aegis_oauth_state", state, 300, "/", "", false, true)
	c.Redirect(http.StatusFound, url)
}

func (h *AuthHandler) callback(c *gin.Context) {
	state := c.Query("state")
	cookie, err := c.Cookie("aegis_oauth_state")
	if err != nil || state == "" || state != cookie {
		WriteError(c, service.ErrInvalidOAuthState())
		return
	}
	token, user, err := h.auth.CompleteLogin(c.Request.Context(), c.Param("provider"), c.Query("code"))
	if err != nil {
		WriteError(c, err)
		return
	}
	c.SetCookie(sessionCookie, token, 0, "/", "", false, true)
	if c.Query("format") == "json" {
		WriteJSON(c, http.StatusOK, service.UserJSON(user))
		return
	}
	redirectURL := h.publicURL
	if redirectURL == "" {
		redirectURL = "/"
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *AuthHandler) logout(c *gin.Context) {
	token, _ := c.Cookie(sessionCookie)
	if err := h.auth.Logout(c.Request.Context(), token); err != nil {
		WriteError(c, err)
		return
	}
	c.SetCookie(sessionCookie, "", -1, "/", "", false, true)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) me(c *gin.Context) {
	token, _ := c.Cookie(sessionCookie)
	user, err := h.auth.CurrentUser(c.Request.Context(), token)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.UserJSON(user))
}

func (h *AuthHandler) patchMe(c *gin.Context) {
	var body struct {
		Locale string `json:"locale"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	token, _ := c.Cookie(sessionCookie)
	user, err := h.auth.UpdateLocale(c.Request.Context(), token, body.Locale)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.UserJSON(user))
}

func SessionToken(c *gin.Context) string {
	token, _ := c.Cookie(sessionCookie)
	if token != "" {
		return token
	}
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
