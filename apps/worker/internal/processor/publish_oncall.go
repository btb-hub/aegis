package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
)

type onCallAnnouncer interface {
	AnnounceOnCall(ctx context.Context, channelID, teamName string, people []integrations.OnCallPerson, locale string) error
}

type PublishOnCallStore interface {
	GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error)
	ListTeamsWithChatChannels(ctx context.Context) ([]db.Team, error)
	CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]db.OnCallUser, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	SetTeamOnCallAnnounced(ctx context.Context, id uuid.UUID, fingerprint string) error
	GetTeamWorkspaceID(ctx context.Context, teamID uuid.UUID) (uuid.UUID, error)
	GetWorkspaceIntegration(ctx context.Context, workspaceID uuid.UUID, kind string) (db.Integration, error)
	GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error)
}

type RotationPublishStore interface {
	ListTeamsWithChatChannels(ctx context.Context) ([]db.Team, error)
	CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]db.OnCallUser, error)
	HasPendingPublishOnCall(ctx context.Context, teamID uuid.UUID) (bool, error)
	EnqueuePublishOnCall(ctx context.Context, teamID uuid.UUID) error
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
		teams, err := p.store.ListTeamsWithChatChannels(ctx)
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
	if !team.HasChatChannel() {
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

	slackAnnouncer, expressAnnouncer, err := p.resolveAnnouncers(ctx, team.ID)
	if err != nil {
		return err
	}

	var slackErr, expressErr error
	if nonemptyPtr(team.SlackChannelID) && slackAnnouncer != nil {
		slackErr = slackAnnouncer.AnnounceOnCall(ctx, *team.SlackChannelID, team.Name, people, locale)
		if slackErr != nil {
			p.log.Error("publish_oncall slack failed", "team_id", team.ID.String(), "error", slackErr)
		}
	}
	if nonemptyPtr(team.ExpressChatID) && expressAnnouncer != nil {
		expressErr = expressAnnouncer.AnnounceOnCall(ctx, *team.ExpressChatID, team.Name, people, locale)
		if expressErr != nil {
			p.log.Error("publish_oncall express failed", "team_id", team.ID.String(), "error", expressErr)
		}
	}

	attemptedSlack := nonemptyPtr(team.SlackChannelID) && slackAnnouncer != nil
	attemptedExpress := nonemptyPtr(team.ExpressChatID) && expressAnnouncer != nil
	if !attemptedSlack && !attemptedExpress {
		return nil
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

func (p *PublishOnCallProcessor) resolveAnnouncers(ctx context.Context, teamID uuid.UUID) (slack onCallAnnouncer, express onCallAnnouncer, err error) {
	if p.announcers != nil {
		return p.announcers(ctx, teamID)
	}
	reg, _, err := loadWorkspaceRegistry(ctx, p.store, teamID, p.publicURL)
	if err != nil {
		return nil, nil, err
	}
	if provider, ok := reg.Chat("slack"); ok {
		slack, _ = provider.(onCallAnnouncer)
	}
	if provider, ok := reg.Chat("express"); ok {
		express, _ = provider.(onCallAnnouncer)
	}
	return slack, express, nil
}

func EnqueueOnCallRotationPublishes(ctx context.Context, store RotationPublishStore, now time.Time) error {
	teams, err := store.ListTeamsWithChatChannels(ctx)
	if err != nil {
		return err
	}
	for _, team := range teams {
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
