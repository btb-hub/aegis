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
