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

	"github.com/aegis/aegis/devtools/alert-simulator/simulator"
)

func main() {
	var (
		once          = flag.Bool("once", false, "send one random alert and exit")
		all           = flag.Bool("all", false, "send every built-in scenario once and exit")
		list          = flag.Bool("list", false, "list scenario ids and exit")
		scenario      = flag.String("scenario", "", "send a specific scenario by id (see -list)")
		interval      = flag.Duration("interval", 0, "repeat interval (overrides ALERT_SIM_INTERVAL)")
		webhook       = flag.String("url", "", "webhook URL (overrides AEGIS_WEBHOOK_URL / PUBLIC_URL)")
		apiBase       = flag.String("api", "", "Aegis API base URL for bootstrap (overrides AEGIS_API_URL / PUBLIC_URL)")
		team          = flag.String("team", "", "override team label for all scenarios (default: per-scenario tier routing)")
		project       = flag.String("project", "", "project label for routing (default: same as team)")
		bootstrap     = flag.Bool("bootstrap", false, "ensure demo routing via Aegis API before sending alerts")
		bootstrapOnly = flag.Bool("bootstrap-only", false, "ensure demo routing via Aegis API and exit")
	)
	flag.Parse()

	if *list {
		for _, s := range simulator.Catalog() {
			routeTeam := simulator.EffectiveRoutingTeam(s, simulator.LabelDefaults{})
			fmt.Printf("%s\t%s\t[%s] team=%s %s\n", s.ID, s.AlertName, s.Severity, routeTeam, s.Summary)
		}
		return
	}

	cfg := simulator.LoadConfig()
	if *webhook != "" {
		cfg.WebhookURL = *webhook
	}
	if *apiBase != "" {
		cfg.APIBaseURL = strings.TrimRight(*apiBase, "/")
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

	if *bootstrapOnly {
		if err := runBootstrap(cfg); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *bootstrap {
		if err := runBootstrap(cfg); err != nil {
			log.Fatal(err)
		}
	}

	if cfg.Secret == "" {
		log.Fatal("WEBHOOK_SECRET is required (set in .env or environment)")
	}

	client := simulator.NewClient(cfg.WebhookURL, cfg.Secret)
	defaults := cfg.LabelDefaults()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sendOne := func(s simulator.Scenario) error {
		routeTeam := simulator.EffectiveRoutingTeam(s, defaults)
		result, err := client.SendScenario(ctx, s, defaults, "")
		if err != nil {
			return fmt.Errorf("%s: %w", s.ID, err)
		}
		log.Printf("sent %s (%s) team=%s → HTTP %d %s", s.ID, s.AlertName, routeTeam, result.StatusCode, strings.TrimSpace(result.Body))
		log.Printf("  → alerts on /alerts; incidents on /incidents after worker routes them (routing rule team=%q)", routeTeam)
		return nil
	}

	switch {
	case *scenario != "":
		s, ok := simulator.ScenarioByID(*scenario)
		if !ok {
			log.Fatalf("unknown scenario %q (use -list)", *scenario)
		}
		if err := sendOne(s); err != nil {
			log.Fatal(err)
		}
	case *all:
		for _, s := range simulator.Catalog() {
			if err := sendOne(s); err != nil {
				log.Fatal(err)
			}
			time.Sleep(200 * time.Millisecond)
		}
	case *once:
		catalog := simulator.Catalog()
		s := catalog[rand.Intn(len(catalog))]
		if err := sendOne(s); err != nil {
			log.Fatal(err)
		}
	default:
		catalog := simulator.Catalog()
		if cfg.Team != "" {
			log.Printf("alert simulator running every %s → %s (all alerts forced to team=%s)", cfg.Interval, cfg.WebhookURL, cfg.Team)
		} else {
			log.Printf("alert simulator running every %s → %s (routing across noc/l1/ops/platform per scenario)", cfg.Interval, cfg.WebhookURL)
		}
		log.Printf("prerequisites: api + worker running; run: make dev-simulator-bootstrap")
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

func runBootstrap(cfg simulator.Config) error {
	api, err := simulator.NewAegisAPI(cfg.APIBaseURL)
	if err != nil {
		return err
	}
	result, err := simulator.EnsureDemoRouting(context.Background(), api, simulator.BootstrapOptions{
		APIBaseURL: cfg.APIBaseURL,
	})
	if err != nil {
		return err
	}
	for _, team := range result.Teams {
		switch {
		case team.CreatedTeam && team.CreatedRule:
			log.Printf("created %s (%s, tier %s) and routing rule %s (team=%q)", team.TeamName, team.TeamID, team.Tier, team.RoutingRuleID, team.TeamLabel)
		case team.CreatedRule:
			log.Printf("created routing rule %s for existing team %s (%s, tier %s, team=%q)", team.RoutingRuleID, team.TeamID, team.TeamName, team.Tier, team.TeamLabel)
		default:
			log.Printf("routing rule exists for team=%q → %s (%s, tier %s)", team.TeamLabel, team.TeamName, team.TeamID, team.Tier)
		}
	}
	if result.CreatedPaths > 0 {
		log.Printf("created %d escalation path(s): NOC→Helpdesk→Ops→Platform", result.CreatedPaths)
	} else {
		log.Printf("escalation paths already configured (NOC→Helpdesk→Ops→Platform)")
	}
	log.Printf("send alerts with: make simulate-alert")
	return nil
}
