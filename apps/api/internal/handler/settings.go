package handler

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const bootstrapCookie = "aegis_installation"

type SettingsHandler struct {
	settings *service.SettingsService
	auth     *service.AuthService
}

func NewSettingsHandler(settings *service.SettingsService, auth *service.AuthService) *SettingsHandler {
	return &SettingsHandler{settings, auth}
}

func (h *SettingsHandler) Register(r gin.IRouter) {
	admin := r.Group("/api/v1/settings", middleware.RequireSession(h.auth), middleware.RequireAdmin())
	admin.Use(func(c *gin.Context) {
		if c.Request.Method != "GET" {
			middleware.RequireSameOrigin(config.PublicURL(c.Request.Context(), ""))(c)
		}
	})
	admin.GET("", h.get)
	admin.PATCH("/:section", h.patch)
	admin.GET("/providers/:provider", h.draft)
	admin.PUT("/providers/:provider", h.saveDraft)
	admin.POST("/providers/:provider/test", h.test)
	admin.POST("/providers/:provider/activate", h.activate)
	r.GET("/api/v1/bootstrap/status", h.bootstrapStatus)
	r.POST("/api/v1/bootstrap/session", h.openBootstrap)
	bootstrap := r.Group("/api/v1/bootstrap", h.requireBootstrap)
	bootstrap.GET("/configuration", h.bootstrapConfiguration)
	bootstrap.GET("/providers/:provider", h.draft)
	bootstrap.PUT("/providers/:provider", h.saveDraft)
	bootstrap.POST("/providers/:provider/test", h.test)
}

func (h *SettingsHandler) get(c *gin.Context) {
	doc, err := h.settings.Repo.GetSettings(c.Request.Context())
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.RedactedSettings(doc))
}

func (h *SettingsHandler) patch(c *gin.Context) {
	var body struct {
		Revision   int64             `json:"revision"`
		Values     map[string]string `json:"values"`
		ConfirmURL bool              `json:"confirm_public_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	actor, _ := middleware.UserFromContext(c)
	doc, err := h.settings.Repo.PatchSettings(c.Request.Context(), actor.ID, body.Revision, c.Param("section"), body.Values, body.ConfirmURL)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.RedactedSettings(doc))
}

func (h *SettingsHandler) draft(c *gin.Context) {
	draft, err := h.settings.Draft(c.Request.Context(), c.Param("provider"))
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.RedactedDraft(draft))
}

func installationRequest(c *gin.Context) bool {
	return strings.HasPrefix(c.FullPath(), "/api/v1/bootstrap/")
}

func (h *SettingsHandler) saveDraft(c *gin.Context) {
	var body struct {
		Revision int64             `json:"revision"`
		Values   map[string]string `json:"values"`
		Email    string            `json:"expected_email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	var actor *uuid.UUID
	if !installationRequest(c) {
		user, _ := middleware.UserFromContext(c)
		actor = &user.ID
		body.Email = user.Email
	}
	draft, err := h.settings.Repo.SaveProviderDraft(c.Request.Context(), actor, c.Param("provider"), body.Revision, body.Values, body.Email)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.RedactedDraft(draft))
}

func (h *SettingsHandler) test(c *gin.Context) {
	kind := "provider_test"
	session := SessionToken(c)
	user, _ := middleware.UserFromContext(c)
	email := user.Email
	if installationRequest(c) {
		kind = "bootstrap"
		session, _ = c.Cookie(bootstrapCookie)
		draft, err := h.settings.Repo.GetProviderDraft(c.Request.Context(), c.Param("provider"))
		if err != nil {
			WriteError(c, err)
			return
		}
		email = draft.ExpectedEmail
	}
	authorizationURL, _, err := h.settings.Start(c.Request.Context(), c.Param("provider"), kind, session, email)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, gin.H{"authorization_url": authorizationURL})
}

func (h *SettingsHandler) activate(c *gin.Context) {
	var body struct {
		Revision      int64 `json:"revision"`
		DraftRevision int64 `json:"draft_revision"`
		Enabled       bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	user, _ := middleware.UserFromContext(c)
	doc, err := h.settings.Repo.ActivateProvider(c.Request.Context(), user.ID, sessiontoken.Hash(SessionToken(c)), c.Param("provider"), body.Revision, body.DraftRevision, body.Enabled)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.RedactedSettings(doc))
}

func (h *SettingsHandler) bootstrapStatus(c *gin.Context) {
	available, err := h.settings.BootstrapAvailable(c.Request.Context())
	if err != nil {
		WriteError(c, err)
		return
	}
	valid := false
	if available {
		token, _ := c.Cookie(bootstrapCookie)
		if token != "" {
			valid, err = h.settings.Repo.ValidBootstrapSession(c.Request.Context(), sessiontoken.Hash(token))
			if err != nil {
				WriteError(c, err)
				return
			}
		}
	}
	WriteJSON(c, http.StatusOK, gin.H{"available": available, "session_active": valid})
}

func (h *SettingsHandler) openBootstrap(c *gin.Context) {
	var body struct {
		Token     string `json:"token"`
		PublicURL string `json:"public_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	origin, err := url.Parse(c.GetHeader("Origin"))
	expected, e := url.Parse(body.PublicURL)
	if err != nil || e != nil || !config.ValidURL(body.PublicURL) || origin.User != nil || origin.Host == "" || !strings.EqualFold(origin.Host, c.Request.Host) || !strings.EqualFold(origin.Host, expected.Host) || origin.Scheme != expected.Scheme || c.GetHeader("Sec-Fetch-Site") == "cross-site" {
		WriteError(c, apperrors.Forbidden("request must originate from Aegis"))
		return
	}
	token, err := h.settings.OpenBootstrap(c.Request.Context(), body.Token, body.PublicURL)
	if err != nil {
		WriteError(c, err)
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(bootstrapCookie, token, 1800, "/", "", expected.Scheme == "https", true)
	WriteJSON(c, http.StatusOK, gin.H{"session_active": true})
}

func (h *SettingsHandler) requireBootstrap(c *gin.Context) {
	token, _ := c.Cookie(bootstrapCookie)
	valid, err := h.settings.Repo.ValidBootstrapSession(c.Request.Context(), sessiontoken.Hash(token))
	if err != nil {
		WriteError(c, err)
		c.Abort()
		return
	}
	if !valid {
		WriteError(c, apperrors.Unauthorized("installation session expired"))
		c.Abort()
		return
	}
	if c.Request.Method != "GET" {
		middleware.RequireSameOrigin(config.PublicURL(c.Request.Context(), ""))(c)
	}
}

func (h *SettingsHandler) bootstrapConfiguration(c *gin.Context) {
	doc, err := h.settings.Repo.GetSettings(c.Request.Context())
	if err != nil {
		WriteError(c, err)
		return
	}
	// Installation access cannot read or mutate behavior, integration or root secrets.
	values := map[string]string{"PUBLIC_URL": doc.Values["PUBLIC_URL"]}
	WriteJSON(c, http.StatusOK, gin.H{"values": values, "revision": doc.Revision})
}

// Login uses database-backed, revision-bound authorizations and verified ID tokens.
func (h *SettingsHandler) Login(c *gin.Context) {
	url, state, err := h.settings.Start(c.Request.Context(), c.Param("provider"), "login", "", "")
	if err != nil {
		c.Redirect(http.StatusFound, "/login?auth_error=unconfigured&provider="+urlEncode(c.Param("provider")))
		return
	}
	secure := strings.HasPrefix(config.PublicURL(c.Request.Context(), ""), "https://")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oauthStateCookie, state, 300, "/", "", secure, true)
	redirect := c.Query("redirect")
	if !isSafeRedirectPath(redirect) {
		redirect = "/"
	}
	c.SetCookie(oauthRedirectCookie, redirect, 300, "/", "", secure, true)
	c.Redirect(http.StatusFound, url)
}

func urlEncode(value string) string { return url.QueryEscape(value) }

func (h *SettingsHandler) Callback(c *gin.Context) {
	state := c.Query("state")
	session := SessionToken(c)
	isLogin := strings.HasPrefix(state, "login.")
	isBootstrap := strings.HasPrefix(state, "bootstrap.")
	target := "/settings?section=authentication&provider=" + urlEncode(c.Param("provider"))
	if isLogin {
		target = "/login"
		session, _ = c.Cookie(oauthStateCookie)
		if state == "" || state != session {
			c.Redirect(http.StatusFound, target+"?settings_error=authorization_failed")
			return
		}
	}
	if isBootstrap {
		target = "/bootstrap?provider=" + urlEncode(c.Param("provider"))
		session, _ = c.Cookie(bootstrapCookie)
	}
	a, info, token, err := h.settings.Complete(c.Request.Context(), c.Param("provider"), state, session, c.Query("code"), c.Query("error"))
	if err == nil && a.Kind == "login" {
		token, _, err = h.auth.CompleteVerifiedLogin(c.Request.Context(), a.Provider, &service.OIDCUserInfo{Sub: info.Subject, Email: info.Email, DisplayName: info.DisplayName, AvatarURL: info.AvatarURL, SlackUserID: info.SlackUserID})
	}
	if err != nil {
		separator := "?"
		if strings.Contains(target, "?") {
			separator = "&"
		}
		c.Redirect(http.StatusFound, target+separator+"settings_error=authorization_failed")
		return
	}
	secure := strings.HasPrefix(config.PublicURL(c.Request.Context(), ""), "https://")
	c.SetSameSite(http.SameSiteLaxMode)
	if a.Kind == "login" || a.Kind == "bootstrap" {
		c.SetCookie(sessionCookie, token, 0, "/", "", secure, true)
	}
	if a.Kind == "login" {
		redirect, _ := c.Cookie(oauthRedirectCookie)
		if !isSafeRedirectPath(redirect) {
			redirect = "/"
		}
		target = withConnectedQuery(redirect, a.Provider)
		c.SetCookie(oauthStateCookie, "", -1, "/", "", secure, true)
		c.SetCookie(oauthRedirectCookie, "", -1, "/", "", secure, true)
		if c.Query("format") == "json" {
			profile, e := h.auth.CurrentUserProfile(c.Request.Context(), token)
			if e != nil {
				WriteError(c, e)
				return
			}
			WriteJSON(c, http.StatusOK, service.UserJSON(profile.User, profile.Identities))
			return
		}
	} else if a.Kind == "bootstrap" {
		c.SetCookie(bootstrapCookie, "", -1, "/", "", secure, true)
		target = "/settings?section=authentication&settings_result=installed"
	} else {
		target += "&settings_result=tested"
	}
	c.Redirect(http.StatusFound, strings.TrimRight(config.PublicURL(c.Request.Context(), ""), "/")+target)
}
