package simulator

import (
	"encoding/json"
	"fmt"
	"time"
)

// Scenario describes a realistic monitoring alert pattern.
type Scenario struct {
	ID          string
	AlertName   string
	Severity    string
	Summary     string
	Description string
	ExtraLabels map[string]string
}

// LabelDefaults are applied to every generated payload (team/project for routing).
type LabelDefaults struct {
	Team    string
	Project string
}

// Payload is the webhook JSON shape accepted by the Aegis alert webhook.
type Payload struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
}

// Catalog returns built-in scenarios mimicking common production issues.
func Catalog() []Scenario {
	return []Scenario{
		{
			ID: "high_cpu", AlertName: "HighCPUUsage", Severity: "critical",
			Summary:     "CPU usage above 90% for 5 minutes",
			Description: "Host cpu_idle is below 10% on api-1. Check recent deploys or traffic spikes.",
			ExtraLabels: map[string]string{"service": "api", "job": "node-exporter"},
		},
		{
			ID: "disk_full", AlertName: "DiskSpaceCritical", Severity: "critical",
			Summary:     "Disk usage above 95% on /var",
			Description: "Root volume on db-primary is nearly full. Log rotation or volume expansion required.",
			ExtraLabels: map[string]string{"service": "postgres", "mountpoint": "/var"},
		},
		{
			ID: "oom_killed", AlertName: "PodOOMKilled", Severity: "warning",
			Summary:     "Container restarted after OOM",
			Description: "Pod payments-worker was OOMKilled in namespace production. Review memory limits.",
			ExtraLabels: map[string]string{"service": "payments-worker", "namespace": "production"},
		},
		{
			ID: "http_5xx", AlertName: "HighHTTP5xxRate", Severity: "critical",
			Summary:     "5xx error rate above 2% for 3 minutes",
			Description: "Ingress gateway returning elevated 502/503 responses to checkout API.",
			ExtraLabels: map[string]string{"service": "checkout-api", "route": "/v1/orders"},
		},
		{
			ID: "db_connections", AlertName: "DatabaseConnectionPoolExhausted", Severity: "critical",
			Summary:     "PostgreSQL connection pool at capacity",
			Description: "All 100 connections in use on platform-db. Long-running queries suspected.",
			ExtraLabels: map[string]string{"service": "postgres", "cluster": "platform-db"},
		},
		{
			ID: "cert_expiry", AlertName: "SSLCertificateExpiringSoon", Severity: "warning",
			Summary:     "TLS certificate expires in less than 7 days",
			Description: "Certificate for api.example.com expires soon. Renew via cert-manager or ACME.",
			ExtraLabels: map[string]string{"service": "ingress", "issuer": "letsencrypt"},
		},
		{
			ID: "memory_pressure", AlertName: "HighMemoryUsage", Severity: "warning",
			Summary:     "Memory usage above 85% for 10 minutes",
			Description: "Redis cache node mem_used exceeds threshold. Eviction policy may be insufficient.",
			ExtraLabels: map[string]string{"service": "redis", "role": "cache"},
		},
		{
			ID: "queue_backlog", AlertName: "MessageQueueBacklog", Severity: "warning",
			Summary:     "Kafka consumer lag above 10k messages",
			Description: "Consumer group billing-events is falling behind. Scale workers or investigate poison messages.",
			ExtraLabels: map[string]string{"service": "kafka", "topic": "billing-events"},
		},
		{
			ID: "replication_lag", AlertName: "PostgresReplicationLag", Severity: "warning",
			Summary:     "Replica lag exceeds 30 seconds",
			Description: "Read replica db-replica-2 is lagging primary. Check network or heavy writes.",
			ExtraLabels: map[string]string{"service": "postgres", "replica": "db-replica-2"},
		},
		{
			ID: "dns_failures", AlertName: "DNSResolutionFailures", Severity: "critical",
			Summary:     "DNS lookup failure rate elevated",
			Description: "CoreDNS timeouts when resolving external payment provider hostnames.",
			ExtraLabels: map[string]string{"service": "coredns", "zone": "cluster.local"},
		},
		{
			ID: "latency_slo", AlertName: "LatencySLOBreach", Severity: "warning",
			Summary:     "P99 latency above 800ms for 5 minutes",
			Description: "Search API p99 latency breached SLO. Trace downstream dependency calls.",
			ExtraLabels: map[string]string{"service": "search-api", "slo": "p99<500ms"},
		},
		{
			ID: "pod_crashloop", AlertName: "PodCrashLooping", Severity: "critical",
			Summary:     "Pod in CrashLoopBackOff for 15 minutes",
			Description: "Deployment auth-service has pods failing readiness after config change.",
			ExtraLabels: map[string]string{"service": "auth-service", "namespace": "production"},
		},
	}
}

// ScenarioByID returns a scenario from Catalog or false.
func ScenarioByID(id string) (Scenario, bool) {
	for _, s := range Catalog() {
		if s.ID == id {
			return s, true
		}
	}
	return Scenario{}, false
}

// BuildPayload creates a unique firing alert payload for the scenario.
func BuildPayload(scenario Scenario, defaults LabelDefaults, instanceSuffix string) Payload {
	if defaults.Team == "" {
		defaults.Team = "platform"
	}
	if defaults.Project == "" {
		defaults.Project = defaults.Team
	}
	if instanceSuffix == "" {
		instanceSuffix = fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}

	labels := map[string]string{
		"alertname": scenario.AlertName,
		"severity":  scenario.Severity,
		"team":      defaults.Team,
		"project":   defaults.Project,
		"instance":  fmt.Sprintf("%s-%s", scenario.ID, instanceSuffix),
	}
	for k, v := range scenario.ExtraLabels {
		labels[k] = v
	}

	return Payload{
		Status: "firing",
		Labels: labels,
		Annotations: map[string]string{
			"summary":     scenario.Summary,
			"description": scenario.Description,
		},
		StartsAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// MarshalPayload encodes the payload as JSON for the webhook body.
func MarshalPayload(p Payload) ([]byte, error) {
	return json.Marshal(p)
}
