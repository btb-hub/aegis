package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	ID              uuid.UUID `json:"id"`
	Provider        string    `json:"provider"`
	ProviderSub     string    `json:"provider_sub"`
	Email           string    `json:"email"`
	DisplayName     string    `json:"display_name"`
	Role            string    `json:"role"`
	Locale          string    `json:"locale"`
	SlackUserID     *string   `json:"slack_user_id"`
	ExpressUserHuid pgtype.UUID `json:"express_user_huid"`
	CreatedAt       time.Time `json:"created_at"`
}

type Session struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"token_hash"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Job struct {
	ID        uuid.UUID       `json:"id"`
	Kind      string          `json:"kind"`
	Payload   []byte          `json:"payload"`
	Status    string          `json:"status"`
	RunAt     time.Time       `json:"run_at"`
	Attempts  int32           `json:"attempts"`
	LastError *string         `json:"last_error"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type Alert struct {
	ID          uuid.UUID `json:"id"`
	Fingerprint string    `json:"fingerprint"`
	Status      string    `json:"status"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Body        *string   `json:"body"`
	Labels      []byte    `json:"labels"`
	SearchTsv   *string   `json:"search_tsv"`
	RawPayload  []byte    `json:"raw_payload"`
	ReceivedAt  time.Time `json:"received_at"`
}

type Team struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TeamMembership struct {
	ID        uuid.UUID `json:"id"`
	TeamID    uuid.UUID `json:"team_id"`
	UserID    uuid.UUID `json:"user_id"`
	TeamRole  string    `json:"team_role"`
	CreatedAt time.Time `json:"created_at"`
}

type TeamMember struct {
	ID          uuid.UUID `json:"id"`
	TeamID      uuid.UUID `json:"team_id"`
	UserID      uuid.UUID `json:"user_id"`
	TeamRole    string    `json:"team_role"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type Schedule struct {
	ID        uuid.UUID `json:"id"`
	TeamID    uuid.UUID `json:"team_id"`
	Name      string    `json:"name"`
	Timezone  string    `json:"timezone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ScheduleLayer struct {
	ID                 uuid.UUID   `json:"id"`
	ScheduleID         uuid.UUID   `json:"schedule_id"`
	Priority           int32       `json:"priority"`
	RotationType       string      `json:"rotation_type"`
	HandoffWeekday     int32       `json:"handoff_weekday"`
	HandoffTime        time.Time   `json:"handoff_time"`
	ParticipantUserIDs []uuid.UUID `json:"participant_user_ids"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

type ScheduleWithLayers struct {
	Schedule Schedule        `json:"schedule"`
	Layers   []ScheduleLayer `json:"layers"`
}

type Override struct {
	ID        uuid.UUID `json:"id"`
	TeamID    uuid.UUID `json:"team_id"`
	UserID    uuid.UUID `json:"user_id"`
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	CreatedAt time.Time `json:"created_at"`
}

type OnCallSlot struct {
	ID        uuid.UUID `json:"id"`
	TeamID    uuid.UUID `json:"team_id"`
	UserID    uuid.UUID `json:"user_id"`
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

type OnCallUser struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Source      string    `json:"source"`
}

type RoutingRule struct {
	ID          uuid.UUID `json:"id"`
	TeamID      uuid.UUID `json:"team_id"`
	MatchLabels []byte    `json:"match_labels"`
	Priority    int32     `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Incident struct {
	ID              uuid.UUID  `json:"id"`
	TeamID          uuid.UUID  `json:"team_id"`
	AssigneeID      *uuid.UUID `json:"assignee_id"`
	Status          string     `json:"status"`
	Severity        string     `json:"severity"`
	Title           string     `json:"title"`
	Fingerprint     string     `json:"fingerprint"`
	JiraIssueKey    *string    `json:"jira_issue_key"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type TimelineEvent struct {
	ID         uuid.UUID  `json:"id"`
	IncidentID uuid.UUID  `json:"incident_id"`
	Kind       string     `json:"kind"`
	ActorID    *uuid.UUID `json:"actor_id"`
	Payload    []byte     `json:"payload"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Integration struct {
	ID        uuid.UUID `json:"id"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	Config    []byte    `json:"config"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Notification struct {
	ID            uuid.UUID  `json:"id"`
	IncidentID    uuid.UUID  `json:"incident_id"`
	IntegrationID uuid.UUID  `json:"integration_id"`
	Status        string     `json:"status"`
	ExternalRef   *string    `json:"external_ref"`
	SentAt        *time.Time `json:"sent_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type ProcessAlertResult struct {
	IncidentID uuid.UUID
	Created    bool
}

type SavedView struct {
	ID        uuid.UUID `json:"id"`
	OwnerID   uuid.UUID `json:"owner_id"`
	Name      string    `json:"name"`
	Filter    []byte    `json:"filter"`
	Shared    bool      `json:"shared"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Handoff struct {
	ID          uuid.UUID  `json:"id"`
	IncidentID  uuid.UUID  `json:"incident_id"`
	FromUserID  *uuid.UUID `json:"from_user_id"`
	ToUserID    *uuid.UUID `json:"to_user_id"`
	FromTeamID  uuid.UUID  `json:"from_team_id"`
	ToTeamID    uuid.UUID  `json:"to_team_id"`
	Reason      *string    `json:"reason"`
	BouncedAt   *time.Time `json:"bounced_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type HandoffStats struct {
	Count                 int     `json:"count"`
	MedianResponseSeconds float64 `json:"median_response_seconds"`
}

type MetricBucket struct {
	BucketStart time.Time `json:"bucket_start"`
	MeanSeconds float64   `json:"mean_seconds"`
	Count       int       `json:"count"`
}

type MetricTimeSeries struct {
	MeanSeconds float64        `json:"mean_seconds"`
	Count       int            `json:"count"`
	Series      []MetricBucket `json:"series"`
}

type NoiseItem struct {
	Fingerprint string `json:"fingerprint"`
	Title       string `json:"title"`
	Count       int    `json:"count"`
}

type NoiseStats struct {
	Items []NoiseItem `json:"items"`
}

type OnCallLoadItem struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	PageCount   int       `json:"page_count"`
}

type OnCallLoadStats struct {
	Items []OnCallLoadItem `json:"items"`
}

type EscalationStats struct {
	TotalIncidents        int     `json:"total_incidents"`
	EscalatedCount        int     `json:"escalated_count"`
	EscalatedPercent      float64 `json:"escalated_percent"`
	MeanSecondsToEscalate float64 `json:"mean_seconds_to_escalate"`
}
