package handler

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
)

const (
	sessionCookie       = "aegis_session"
	oauthStateCookie    = "aegis_oauth_state"
	oauthRedirectCookie = "aegis_oauth_redirect"
	oauthCookieTTL      = 300
)

type AuthHandler struct {
	auth      *service.AuthService
	publicURL string
}

func NewAuthHandler(auth *service.AuthService, publicURL string) *AuthHandler {
	return &AuthHandler{auth: auth, publicURL: publicURL}
}

func (h *AuthHandler) Register(r gin.IRouter) {
	r.GET("/auth/dev/status", h.devStatus)
	r.GET("/auth/dev/login", h.devLogin)
	r.GET("/auth/providers", h.providers)
	r.GET("/auth/:provider/login", h.login)
	r.GET("/auth/:provider/callback", h.callback)
	r.POST("/auth/logout", h.logout)
	r.GET("/auth/me", h.me)
	r.PATCH("/auth/me", h.patchMe)
}

func (h *AuthHandler) devStatus(c *gin.Context) {
	WriteJSON(c, http.StatusOK, gin.H{"enabled": h.auth.DevAuthEnabled()})
}

func (h *AuthHandler) devLogin(c *gin.Context) {
	if !h.auth.DevAuthEnabled() {
		c.Status(http.StatusNotFound)
		return
	}
	token, _, err := h.auth.DevLogin(c.Request.Context(), c.Query("role"))
	if err != nil {
		c.Redirect(http.StatusFound, h.devLoginFailureURL())
		return
	}
	c.SetCookie(sessionCookie, token, 0, "/", "", false, true)
	c.Redirect(http.StatusFound, h.devRedirectURL(c))
}

func (h *AuthHandler) devLoginFailureURL() string {
	base := strings.TrimRight(h.publicURL, "/")
	if base == "" {
		return "/login?dev_auth_error=1"
	}
	return base + "/login?dev_auth_error=1"
}

func isSafeRedirectPath(path string) bool {
	return path != "" && strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//")
}

func (h *AuthHandler) appRedirectURL(redirectPath string) string {
	base := strings.TrimRight(h.publicURL, "/")
	if isSafeRedirectPath(redirectPath) {
		if base == "" {
			return redirectPath
		}
		return base + redirectPath
	}
	if h.publicURL == "" {
		return "/"
	}
	return h.publicURL
}

func (h *AuthHandler) devRedirectURL(c *gin.Context) string {
	return h.appRedirectURL(c.Query("redirect"))
}

func (h *AuthHandler) providers(c *gin.Context) {
	WriteJSON(c, http.StatusOK, gin.H{"providers": h.auth.ConfiguredProviders()})
}

func (h *AuthHandler) login(c *gin.Context) {
	provider := c.Param("provider")
	loginURL, state, err := h.auth.LoginURL(provider)
	if err != nil {
		c.Redirect(http.StatusFound, h.unconfiguredProviderURL(provider))
		return
	}
	c.SetCookie(oauthStateCookie, state, oauthCookieTTL, "/", "", false, true)
	redirectPath := c.Query("redirect")
	if isSafeRedirectPath(redirectPath) {
		c.SetCookie(oauthRedirectCookie, redirectPath, oauthCookieTTL, "/", "", false, true)
	} else {
		c.SetCookie(oauthRedirectCookie, "", -1, "/", "", false, true)
	}
	c.Redirect(http.StatusFound, loginURL)
}

func (h *AuthHandler) unconfiguredProviderURL(provider string) string {
	query := url.Values{}
	query.Set("auth_error", "unconfigured")
	query.Set("provider", provider)
	base := strings.TrimRight(h.publicURL, "/")
	if base == "" {
		return "/login?" + query.Encode()
	}
	return base + "/login?" + query.Encode()
}

func (h *AuthHandler) callback(c *gin.Context) {
	state := c.Query("state")
	cookie, err := c.Cookie(oauthStateCookie)
	if err != nil || state == "" || state != cookie {
		WriteError(c, service.ErrInvalidOAuthState())
		return
	}
	provider := c.Param("provider")
	token, user, err := h.auth.CompleteLogin(c.Request.Context(), provider, c.Query("code"))
	if err != nil {
		WriteError(c, err)
		return
	}
	c.SetCookie(sessionCookie, token, 0, "/", "", false, true)
	if c.Query("format") == "json" {
		profile, err := h.auth.CurrentUserProfile(c.Request.Context(), token)
		if err != nil {
			WriteJSON(c, http.StatusOK, service.UserJSON(user, nil))
			return
		}
		WriteJSON(c, http.StatusOK, service.UserJSON(profile.User, profile.Identities))
		return
	}
	redirectPath, _ := c.Cookie(oauthRedirectCookie)
	c.SetCookie(oauthRedirectCookie, "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, h.appRedirectURL(withConnectedQuery(redirectPath, provider)))
}

func withConnectedQuery(redirectPath, provider string) string {
	if !isSafeRedirectPath(redirectPath) {
		return redirectPath
	}
	path, query, found := strings.Cut(redirectPath, "?")
	if path != "/account" {
		return redirectPath
	}
	if found {
		return path + "?" + query + "&connected=" + provider
	}
	return path + "?connected=" + provider
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
	profile, err := h.auth.CurrentUserProfile(c.Request.Context(), token)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.UserJSON(profile.User, profile.Identities))
}

func (h *AuthHandler) patchMe(c *gin.Context) {
	var body struct {
		Locale      *string `json:"locale"`
		DisplayName *string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	if body.Locale == nil && body.DisplayName == nil {
		WriteError(c, apperrors.Validation("locale or display_name is required", nil))
		return
	}
	token, _ := c.Cookie(sessionCookie)
	user, err := h.auth.UpdateProfile(c.Request.Context(), token, service.UpdateProfileInput{
		DisplayName: body.DisplayName,
		Locale:      body.Locale,
	})
	if err != nil {
		WriteError(c, err)
		return
	}
	profile, err := h.auth.CurrentUserProfile(c.Request.Context(), token)
	if err != nil {
		WriteJSON(c, http.StatusOK, service.UserJSON(user, nil))
		return
	}
	WriteJSON(c, http.StatusOK, service.UserJSON(profile.User, profile.Identities))
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
