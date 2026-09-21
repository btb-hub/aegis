package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	intexpress "github.com/aegis/aegis/pkg/integrations/express"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type onCallAnnouncer interface {
	AnnounceOnCall(ctx context.Context, channelID, slackUserGroupID, teamName string, people []integrations.OnCallPerson, locale string) error
}

type PublishOnCallStore interface {
	GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error)
	ListTeams(ctx context.Context) ([]db.Team, error)
	CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]db.OnCallUser, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	SetTeamOnCallAnnounced(ctx context.Context, id uuid.UUID, fingerprint string) error
	GetTeamWorkspaceID(ctx context.Context, teamID uuid.UUID) (uuid.UUID, error)
	GetWorkspaceIntegration(ctx context.Context, workspaceID uuid.UUID, kind string) (db.Integration, error)
	GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error)
}

type RotationPublishStore interface {
	ListTeams(ctx context.Context) ([]db.Team, error)
	CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]db.OnCallUser, error)
	HasPendingPublishOnCall(ctx context.Context, teamID uuid.UUID) (bool, error)
	EnqueuePublishOnCall(ctx context.Context, teamID uuid.UUID) error
	GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error)
}

type globalExpressIntegrationStore interface {
	GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error)
}

type PublishOnCallProcessor struct {
	log        *slog.Logger
	store      PublishOnCallStore
	publicURL  string
	announcers func(ctx context.Context, teamID uuid.UUID) (slack onCallAnnouncer, express onCallAnnouncer, err error)
}

func NewPublishOnCallProcessor(log *slog.Logger, store PublishOnCallStore, publicURL string) *PublishOnCallProcessor {
	if log == nil {
		log = slog.Default()
	}
	return &PublishOnCallProcessor{log: log, store: store, publicURL: publicURL}
}

func (p *PublishOnCallProcessor) Handle(ctx context.Context, job Job) error {
	var payload struct {
		TeamID string `json:"team_id"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	if payload.TeamID == "" {
		teams, err := p.store.ListTeams(ctx)
		if err != nil {
			return err
		}
		for _, team := range teams {
			if err := p.publishTeam(ctx, team); err != nil {
				return err
			}
		}
		p.log.Info("publish_oncall nightly", "teams", len(teams))
		return nil
	}
	teamID, err := uuid.Parse(payload.TeamID)
	if err != nil {
		return fmt.Errorf("invalid team_id: %w", err)
	}
	team, err := p.store.GetTeam(ctx, teamID)
	if err != nil {
		return err
	}
	return p.publishTeam(ctx, team)
}

func (p *PublishOnCallProcessor) publishTeam(ctx context.Context, team db.Team) error {
	slackChannelID := stringValue(team.SlackChannelID)
	slackUserGroupID := stringValue(team.SlackUserGroupID)
	slackAnnouncer, expressAnnouncer, expressChatID, err := p.resolveAnnouncers(ctx, team.ID, slackChannelID)
	if err != nil {
		return err
	}

	attemptedSlack := slackChannelID != "" && slackAnnouncer != nil
	attemptedExpress := expressChatID != "" && expressAnnouncer != nil
	if !attemptedSlack && !attemptedExpress {
		return nil
	}

	onCall, err := p.store.CurrentOnCallUsers(ctx, team.ID, time.Now().UTC())
	if err != nil {
		return err
	}
	people := make([]integrations.OnCallPerson, 0, len(onCall))
	locale := "en"
	for _, user := range onCall {
		full, err := p.store.GetUserByID(ctx, user.UserID)
		if err != nil {
			p.log.Error("publish_oncall skip user", "user_id", user.UserID.String(), "error", err)
			people = append(people, integrations.OnCallPerson{DisplayName: user.DisplayName})
			continue
		}
		if full.Locale != "" {
			locale = full.Locale
		}
		people = append(people, integrations.OnCallPerson{
			DisplayName:     full.DisplayName,
			Locale:          full.Locale,
			SlackUserID:     full.SlackUserID,
			ExpressUserHuid: db.ExpressHuidString(full),
		})
	}

	var slackErr, expressErr error
	if attemptedSlack {
		slackErr = slackAnnouncer.AnnounceOnCall(ctx, slackChannelID, slackUserGroupID, team.Name, people, locale)
		if slackErr != nil {
			p.log.Error("publish_oncall slack failed", "team_id", team.ID.String(), "error", slackErr)
		}
	}
	if attemptedExpress {
		expressErr = expressAnnouncer.AnnounceOnCall(ctx, expressChatID, "", team.Name, people, locale)
		if expressErr != nil {
			p.log.Error("publish_oncall express failed", "team_id", team.ID.String(), "error", expressErr)
		}
	}

	if attemptedSlack && slackErr != nil && attemptedExpress && expressErr != nil {
		return fmt.Errorf("publish_oncall both providers failed: slack: %v; express: %v", slackErr, expressErr)
	}
	if attemptedSlack && slackErr != nil && !attemptedExpress {
		return slackErr
	}
	if attemptedExpress && expressErr != nil && !attemptedSlack {
		return expressErr
	}

	return p.store.SetTeamOnCallAnnounced(ctx, team.ID, onCallFingerprint(onCall))
}

func (p *PublishOnCallProcessor) resolveAnnouncers(ctx context.Context, teamID uuid.UUID, slackChannelID string) (slack onCallAnnouncer, express onCallAnnouncer, expressChatID string, err error) {
	if p.announcers != nil {
		slack, express, err = p.announcers(ctx, teamID)
		if err != nil {
			return nil, nil, "", err
		}
	} else if slackChannelID != "" {
		reg, _, err := loadWorkspaceRegistry(ctx, p.store, teamID, p.publicURL, "slack")
		if err != nil {
			return nil, nil, "", err
		}
		if provider, ok := reg.Chat("slack"); ok {
			slack, _ = provider.(onCallAnnouncer)
		}
	}

	globalExpress, expressChatID, err := globalExpressOnCallDestination(ctx, p.store)
	if err != nil {
		p.log.Error("publish_oncall global express unavailable", "team_id", teamID.String(), "error", err)
		return slack, nil, "", nil
	}
	if expressChatID == "" {
		return slack, nil, "", nil
	}
	if express != nil {
		return slack, express, expressChatID, nil
	}
	expressProvider, err := intexpress.NewFromJSON(globalExpress.Config)
	if err != nil {
		p.log.Error("publish_oncall global express initialization failed", "team_id", teamID.String(), "error", err)
		return slack, nil, "", nil
	}
	return slack, expressProvider, expressChatID, nil
}

func globalExpressOnCallDestination(ctx context.Context, store globalExpressIntegrationStore) (db.Integration, string, error) {
	globalExpress, err := store.GetIntegrationByKind(ctx, "express")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Integration{}, "", nil
		}
		return db.Integration{}, "", err
	}
	if !globalExpress.Enabled {
		return db.Integration{}, "", nil
	}
	var expressConfig struct {
		OnCallGroupChatID string `json:"oncall_group_chat_id"`
	}
	if err := json.Unmarshal(globalExpress.Config, &expressConfig); err != nil {
		return db.Integration{}, "", fmt.Errorf("decode global express config: %w", err)
	}
	chatID := strings.TrimSpace(expressConfig.OnCallGroupChatID)
	if chatID != "" {
		if _, err := intexpress.NewFromJSON(globalExpress.Config); err != nil {
			return db.Integration{}, "", fmt.Errorf("load global express provider: %w", err)
		}
	}
	return globalExpress, chatID, nil
}

func EnqueueOnCallRotationPublishes(ctx context.Context, store RotationPublishStore, now time.Time) error {
	teams, err := store.ListTeams(ctx)
	if err != nil {
		return err
	}
	_, expressChatID, err := globalExpressOnCallDestination(ctx, store)
	if err != nil {
		slog.Error("rotation publish global express unavailable", "error", err)
		expressChatID = ""
	}
	for _, team := range teams {
		if stringValue(team.SlackChannelID) == "" && expressChatID == "" {
			continue
		}
		users, err := store.CurrentOnCallUsers(ctx, team.ID, now)
		if err != nil {
			return err
		}
		fp := onCallFingerprint(users)
		if team.OnCallAnnouncedUserIDs != nil && *team.OnCallAnnouncedUserIDs == fp {
			continue
		}
		if fp == "" && (team.OnCallAnnouncedUserIDs == nil || *team.OnCallAnnouncedUserIDs == "") {
			continue
		}
		pending, err := store.HasPendingPublishOnCall(ctx, team.ID)
		if err != nil {
			return err
		}
		if pending {
			continue
		}
		if err := store.EnqueuePublishOnCall(ctx, team.ID); err != nil {
			return err
		}
	}
	return nil
}

func onCallFingerprint(users []db.OnCallUser) string {
	ids := make([]string, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.UserID.String())
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func nonemptyPtr(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}

func stringValue(value *string) string {
	if !nonemptyPtr(value) {
		return ""
	}
	return strings.TrimSpace(*value)
}
