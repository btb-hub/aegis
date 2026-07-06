package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aegis/aegis/apps/api/internal/handler"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/i18n"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db pool: %v", err)
	}
	defer pool.Close()

	if err := loadI18n(); err != nil {
		log.Fatalf("i18n: %v", err)
	}
	if cfg.DevAuthEnabled {
		log.Printf("WARNING: DEV_AUTH_ENABLED — dev login active; never use in production")
	}

	store := db.NewStore(pool)
	auth := service.NewAuthService(cfg, store, store, service.NewOAuthTokenExchanger(cfg))
	alerts := service.NewAlertService(cfg.WebhookSecret, cfg.AlertFingerprintLabels, store)
	health := service.NewHealthService(store)
	teams := service.NewTeamService(store)
	workspaces := service.NewWorkspaceService(store)
	escalation := service.NewEscalationService(store)
	schedules := service.NewScheduleService(store)
	overrides := service.NewOverrideService(store)
	oncall := service.NewOnCallService(store)
	routingRules := service.NewRoutingService(store)
	incidents := service.NewIncidentService(store)
	handoffs := service.NewHandoffService(store)
	analytics := service.NewAnalyticsService(store)
	integrationsSvc := service.NewIntegrationService(store, cfg.PublicURL)
	expressLinks := service.NewExpressLinkService(store)
	savedViews := service.NewSavedViewService(store)
	users := service.NewUserService(store)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	handler.NewHealthHandler(health).Register(r)
	handler.NewAuthHandler(auth, cfg.PublicURL).Register(r)
	handler.NewAlertHandler(alerts, teams, auth).Register(r)
	handler.NewTeamHandler(teams, auth).Register(r)
	handler.NewWorkspaceHandler(workspaces, auth).Register(r)
	handler.NewEscalationHandler(escalation, auth).Register(r)
	handler.NewUserHandler(users, auth).Register(r)
	handler.NewScheduleHandler(schedules, auth).Register(r)
	handler.NewOverrideHandler(overrides, auth).Register(r)
	handler.NewOnCallHandler(oncall, auth).Register(r)
	handler.NewRoutingHandler(routingRules, auth).Register(r)
	handler.NewIncidentHandler(incidents, handoffs, auth).Register(r)
	handler.NewIntegrationHandler(integrationsSvc, auth).Register(r)
	handler.NewSavedViewHandler(savedViews, auth).Register(r)
	handler.NewAnalyticsHandler(analytics, alerts, handoffs, auth).Register(r)
	handler.NewSlackCallbackHandler(incidents, cfg.SlackSigningSecret()).Register(r)
	handler.NewExpressCallbackHandler(incidents, expressLinks, integrationsSvc).Register(r)
	handler.NewExpressLinkHandler(expressLinks, auth).Register(r)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r}
	go func() {
		log.Printf("api listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func loadI18n() error {
	dir := filepath.Join("pkg", "i18n", "messages")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		dir = filepath.Join("..", "..", "pkg", "i18n", "messages")
	}
	return i18n.LoadMessages(dir)
}
