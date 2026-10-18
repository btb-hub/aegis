package handler

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/pkg/anomalymodule"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AnomalyRepository interface {
	ListWorkspaces(context.Context) ([]db.Workspace, error)
	AcceptAnomalyEvent(context.Context, anomalymodule.Event) (db.AnomalyEventResult, error)
}

type AnomalyHandler struct {
	config       *config.Config
	repo         AnomalyRepository
	auth         middleware.SessionUserResolver
	entitlements anomalymodule.EntitlementProvider
	client       *http.Client
}

func NewAnomalyHandler(cfg *config.Config, repo AnomalyRepository, auth middleware.SessionUserResolver, providers ...anomalymodule.EntitlementProvider) *AnomalyHandler {
	entitlement := anomalymodule.EntitlementProvider(anomalymodule.ConfiguredEntitlement{})
	if len(providers) > 0 && providers[0] != nil {
		entitlement = providers[0]
	}
	return &AnomalyHandler{config: cfg, repo: repo, auth: auth, entitlements: entitlement, client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (h *AnomalyHandler) Register(r gin.IRouter) {
	r.GET("/api/v1/modules/anomaly-detector/capabilities", middleware.RequireSession(h.auth), h.capabilities)
	r.GET("/api/v1/modules/anomaly-detector/entitlement", h.serviceAuth, h.entitlement)
	r.POST("/api/v1/modules/anomaly-detector/events", h.serviceAuth, h.events)
	browser := r.Group(anomalymodule.Prefix, middleware.RequireBrowserSession(h.auth))
	browser.Any("", h.proxy)
	browser.Any("/*path", h.proxy)
}
func (h *AnomalyHandler) settings(c *gin.Context) config.AnomalyConfig {
	return config.Runtime(c.Request.Context(), h.config).Anomaly
}
func (h *AnomalyHandler) serviceAuth(c *gin.Context) {
	cfg := h.settings(c)
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !cfg.Configured() || c.GetHeader("Authorization") == token || subtle.ConstantTimeCompare([]byte(token), []byte(cfg.ServiceToken)) != 1 {
		WriteError(c, apperrors.Unauthorized("invalid module service credentials"))
		c.Abort()
		return
	}
	c.Next()
}
func (h *AnomalyHandler) entitlement(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	WriteJSON(c, http.StatusOK, h.entitlements.Grant(h.settings(c), time.Now().UTC()))
}
func (h *AnomalyHandler) capabilities(c *gin.Context) {
	status, _ := anomalymodule.Check(c.Request.Context(), h.settings(c), h.client)
	status.Enabled = h.entitlements.Grant(h.settings(c), time.Now()).Enabled
	c.Header("Cache-Control", "no-store")
	WriteJSON(c, http.StatusOK, status)
}
func (h *AnomalyHandler) events(c *gin.Context) {
	cfg := h.settings(c)
	if !h.entitlements.Grant(cfg, time.Now()).Enabled {
		WriteError(c, apperrors.Forbidden("anomaly detection access is disabled"))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var event anomalymodule.Event
	if err := decoder.Decode(&event); err != nil {
		WriteError(c, apperrors.Validation("invalid anomaly event JSON", nil))
		return
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		WriteError(c, apperrors.Validation("send exactly one anomaly event", nil))
		return
	}
	if event.InstallationID != cfg.InstallationID {
		WriteError(c, apperrors.Forbidden("event installation does not match"))
		return
	}
	if err := event.Validate(time.Now().UTC()); err != nil {
		WriteError(c, apperrors.Validation(err.Error(), nil))
		return
	}
	result, err := h.repo.AcceptAnomalyEvent(c.Request.Context(), event)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusAccepted, result)
}
func (h *AnomalyHandler) proxy(c *gin.Context) {
	cfg := h.settings(c)
	user, _ := middleware.UserFromContext(c)
	read := c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions
	if !read {
		middleware.RequireSameOrigin(config.PublicURL(c.Request.Context(), h.config.PublicURL))(c)
		if c.IsAborted() {
			return
		}
		parts := strings.Split(strings.TrimPrefix(c.Request.URL.Path, anomalymodule.Prefix+"/api/v1/history/"), "/")
		feedback := false
		if c.Request.Method == http.MethodPost && strings.HasPrefix(c.Request.URL.Path, anomalymodule.Prefix+"/api/v1/history/") && len(parts) == 2 && parts[1] == "feedback" {
			id, err := uuid.Parse(parts[0])
			feedback = err == nil && id != uuid.Nil && id.String() == parts[0]
		}
		if user.Role != "admin" && !(user.Role == "member" && feedback) {
			WriteError(c, apperrors.Forbidden("this action is not allowed for your role"))
			return
		}
		if !h.entitlements.Grant(cfg, time.Now()).Enabled {
			WriteError(c, apperrors.Forbidden("anomaly detection access is disabled"))
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
		if err != nil {
			WriteError(c, apperrors.Validation("module request exceeds 1 MiB or could not be read", nil))
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		c.Request.ContentLength = int64(len(body))
	}
	status, err := anomalymodule.Check(c.Request.Context(), cfg, h.client)
	if err != nil {
		h.unavailable(c, status)
		return
	}
	workspaces, err := h.repo.ListWorkspaces(c.Request.Context())
	if err != nil {
		WriteError(c, err)
		return
	}
	ids := make([]string, 0, len(workspaces))
	for _, workspace := range workspaces {
		ids = append(ids, workspace.ID.String())
	}
	identity := anomalymodule.SignIdentity(cfg, user.ID.String(), user.Role, user.Locale, ids, h.entitlements.Grant(cfg, time.Now()).Enabled, time.Now().UTC())
	upstream, err := url.Parse(cfg.URL)
	if err != nil {
		h.unavailable(c, status)
		return
	}
	proxy := &httputil.ReverseProxy{Transport: h.client.Transport, FlushInterval: -1}
	proxy.Rewrite = func(out *httputil.ProxyRequest) {
		out.SetURL(upstream)
		out.Out.Host = out.In.Host
		req := out.Out
		for key := range req.Header {
			lower := strings.ToLower(key)
			if strings.HasPrefix(lower, "x-aegis-") || strings.HasPrefix(lower, "x-forwarded-") || lower == "forwarded" || lower == "authorization" || lower == "cookie" || lower == "proxy-authorization" {
				req.Header.Del(key)
			}
		}
		req.Header.Set("X-Aegis-Identity", identity)
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Set-Cookie")
		resp.Header.Del("X-Aegis-Identity")
		resp.Header.Set("Cache-Control", "no-store")
		return nil
	}
	proxy.ErrorHandler = func(http.ResponseWriter, *http.Request, error) { h.unavailable(c, status) }
	timeout := 10 * time.Second
	parts := strings.Split(strings.TrimPrefix(c.Request.URL.Path, anomalymodule.Prefix+"/api/v1/"), "/")
	if c.Request.Method == http.MethodPost && len(parts) == 3 && ((parts[0] == "metrics" && parts[2] == "preview") || (parts[0] == "sources" && parts[2] == "test")) {
		// Source requests are bounded at 120 seconds. Leave room for detection
		// and serialization while the private service admits preview concurrency.
		timeout = 150 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	proxy.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
}

var anomalyUnavailableTemplate = template.Must(template.New("anomaly-unavailable").Parse(`<!doctype html><html lang="{{.Locale}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}} · Aegis</title><style>body{font:16px/1.5 sans-serif;background:#fafafa;color:#18181b;margin:0;padding:32px}main{max-width:680px;margin:80px auto;background:white;border:1px solid #e4e4e7;border-radius:8px;padding:32px}h1{font-size:24px}a{color:#2563eb;display:inline-block;margin:12px 24px 0 0}</style></head><body><main><h1>{{.Title}}</h1><p>{{.Body}}</p>{{if .Admin}}<a href="/settings?section=anomaly">{{.Settings}}</a>{{else}}<p>{{.Ask}}</p>{{end}}<a href="/shifts">{{.Back}}</a></main></body></html>`))

func (h *AnomalyHandler) unavailable(c *gin.Context, status anomalymodule.Status) {
	code := "ANOMALY_MODULE_UNAVAILABLE"
	if !status.Configured {
		code = "ANOMALY_MODULE_NOT_CONFIGURED"
	} else if status.Available && !status.Compatible {
		code = "ANOMALY_MODULE_INCOMPATIBLE"
	}
	if strings.Contains(c.Request.URL.Path, "/api/") || !strings.Contains(c.GetHeader("Accept"), "text/html") {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": code, "message": "Anomaly detection is unavailable. Check module settings and connection.", "details": nil})
		return
	}
	user, _ := middleware.UserFromContext(c)
	copy := map[string]string{"Locale": "en", "Title": "Anomaly detection is unavailable", "Body": "Check the module connection and supported version in Settings. The rest of Aegis is available.", "Settings": "Module settings", "Back": "Back to Aegis"}
	if !status.Configured {
		copy["Body"] = "Configure the anomaly detection module in Settings to connect your installation."
	}
	if user.Locale == "ru" {
		copy = map[string]string{"Locale": "ru", "Title": "Обнаружение аномалий недоступно", "Body": "Проверьте подключение модуля и поддерживаемую версию в настройках. Остальные функции Aegis доступны.", "Settings": "Настройки модуля", "Back": "Вернуться в Aegis"}
		if !status.Configured {
			copy["Body"] = "Настройте модуль обнаружения аномалий, чтобы подключить его к Aegis."
		}
	}
	if user.Role == "admin" {
		copy["Admin"] = "true"
	}
	copy["Ask"] = "Ask an Aegis administrator to check the module connection."
	if user.Locale == "ru" {
		copy["Ask"] = "Попросите администратора Aegis проверить подключение модуля."
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusServiceUnavailable)
	_ = anomalyUnavailableTemplate.Execute(c.Writer, copy)
}
