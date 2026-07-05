package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aegis/aegis/pkg/alertsim"
)

func main() {
	var (
		once      = flag.Bool("once", false, "send one random alert and exit")
		all       = flag.Bool("all", false, "send every built-in scenario once and exit")
		list      = flag.Bool("list", false, "list scenario ids and exit")
		scenario  = flag.String("scenario", "", "send a specific scenario by id (see -list)")
		interval  = flag.Duration("interval", 0, "repeat interval (overrides ALERT_SIM_INTERVAL)")
		webhook   = flag.String("url", "", "webhook URL (overrides AEGIS_WEBHOOK_URL / PUBLIC_URL)")
		team      = flag.String("team", "", "team label for routing (default: platform)")
		project   = flag.String("project", "", "project label for routing (default: same as team)")
	)
	flag.Parse()

	if *list {
		for _, s := range alertsim.Catalog() {
			fmt.Printf("%s\t%s\t[%s] %s\n", s.ID, s.AlertName, s.Severity, s.Summary)
		}
		return
	}

	cfg := alertsim.LoadConfig()
	if *webhook != "" {
		cfg.WebhookURL = *webhook
	}
	if *team != "" {
		cfg.Team = *team
	}
	if *project != "" {
		cfg.Project = *project
	}
	if *interval > 0 {
		cfg.Interval = *interval
	}
	if cfg.Secret == "" {
		log.Fatal("WEBHOOK_SECRET is required (set in .env or environment)")
	}

	client := alertsim.NewClient(cfg.WebhookURL, cfg.Secret)
	defaults := cfg.LabelDefaults()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sendOne := func(s alertsim.Scenario) error {
		result, err := client.SendScenario(ctx, s, defaults, "")
		if err != nil {
			return fmt.Errorf("%s: %w", s.ID, err)
		}
		log.Printf("sent %s (%s) → HTTP %d %s", s.ID, s.AlertName, result.StatusCode, strings.TrimSpace(result.Body))
		log.Printf("  → alerts appear on /alerts; incidents on /incidents after worker routes them (needs routing rule team=%q)", cfg.Team)
		return nil
	}

	switch {
	case *scenario != "":
		s, ok := alertsim.ScenarioByID(*scenario)
		if !ok {
			log.Fatalf("unknown scenario %q (use -list)", *scenario)
		}
		if err := sendOne(s); err != nil {
			log.Fatal(err)
		}
	case *all:
		for _, s := range alertsim.Catalog() {
			if err := sendOne(s); err != nil {
				log.Fatal(err)
			}
			time.Sleep(200 * time.Millisecond)
		}
	case *once:
		catalog := alertsim.Catalog()
		s := catalog[rand.Intn(len(catalog))]
		if err := sendOne(s); err != nil {
			log.Fatal(err)
		}
	default:
		catalog := alertsim.Catalog()
		log.Printf("alert simulator running every %s → %s (team=%s project=%s)", cfg.Interval, cfg.WebhookURL, cfg.Team, cfg.Project)
		log.Printf("prerequisites: api + worker running; routing rule matching team=%q (run: make seed-demo)", cfg.Team)
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()
		for {
			s := catalog[rand.Intn(len(catalog))]
			if err := sendOne(s); err != nil {
				log.Printf("send failed: %v", err)
			}
			select {
			case <-ctx.Done():
				log.Printf("stopped")
				return
			case <-ticker.C:
			}
		}
	}
}
