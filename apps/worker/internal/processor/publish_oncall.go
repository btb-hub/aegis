package processor

import (
	"context"
	"crypto/sha256"
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
		TeamID         string `json:"team_id"`
		PublicationKey string `json:"publication_key"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	if payload.PublicationKey == "" && job.ID != "" {
		payload.PublicationKey = "manual:" + job.ID
	}
	if payload.TeamID == "" {
		teams, err := p.store.ListTeams(ctx)
		if err != nil {
			return err
		}
		var failures []error
		for _, team := range teams {
			if err := p.publishTeam(ctx, team, payload.PublicationKey); err != nil {
				failures = append(failures, err)
			}
		}
		if len(failures) > 0 {
			return errors.Join(failures...)
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
	return p.publishTeam(ctx, team, payload.PublicationKey)
}

func (p *PublishOnCallProcessor) publishTeam(ctx context.Context, team db.Team, keys ...string) error {
	publicationKey := ""
	if len(keys) > 0 {
		publicationKey = keys[0]
	}
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
			people = append(people, integrations.OnCallPerson{DisplayName: user.DisplayName, StartAt: user.StartAt, EndAt: user.EndAt})
			continue
		}
		if full.Locale != "" {
			locale = full.Locale
		}
		people = append(people, integrations.OnCallPerson{
			DisplayName: full.DisplayName,
			StartAt:     user.StartAt, EndAt: user.EndAt,
			Locale:          full.Locale,
			SlackUserID:     full.SlackUserID,
			ExpressUserHuid: db.ExpressHuidString(full),
		})
	}

	var slackErr, expressErr error
	if attemptedSlack {
		slackErr = p.sendOnCall(ctx, team.ID, "slack", slackChannelID, onCallFingerprint(onCall), publicationKey, func() error {
			return slackAnnouncer.AnnounceOnCall(ctx, slackChannelID, slackUserGroupID, team.Name, people, locale)
		})
		if slackErr != nil {
			p.log.Error("publish_oncall slack failed", "team_id", team.ID.String(), "error", slackErr)
		}
	}
	if attemptedExpress {
		var outgoingRef, generation string
		expressErr = p.sendOnCall(ctx, team.ID, "express", expressChatID, onCallFingerprint(onCall), publicationKey+":"+onCallFingerprint(onCall), func() error {
			if tracked, ok := expressAnnouncer.(interface {
				AnnounceOnCallWithRef(context.Context, string, string, string, []integrations.OnCallPerson, string) (string, error)
			}); ok {
				var err error
				outgoingRef, err = tracked.AnnounceOnCallWithRef(ctx, expressChatID, "", team.Name, people, locale)
				return err
			}
			return expressAnnouncer.AnnounceOnCall(ctx, expressChatID, "", team.Name, people, locale)
		}, &generation, &outgoingRef)
		_, atomicRef := p.store.(interface {
			RunOnCallDeliveryWithRef(context.Context, db.PublicationAttempt, uuid.UUID, func() (string, error)) error
		})
		if outgoingRef != "" && !atomicRef {
			if recorder, ok := p.store.(interface {
				RecordOnCallExpressRef(context.Context, uuid.UUID, uuid.UUID, string, string, string) error
			}); ok {
				connector, err := p.store.GetIntegrationByKind(ctx, "express")
				if err == nil {
					err = recorder.RecordOnCallExpressRef(ctx, connector.ID, team.ID, expressChatID, generation, outgoingRef)
				}
				if err != nil {
					expressErr = err
				}
			}
		}
		if expressErr != nil {
			p.log.Error("publish_oncall express failed", "team_id", team.ID.String(), "error", expressErr)
		}
	}

	if attemptedSlack && slackErr != nil && attemptedExpress && expressErr != nil {
		return fmt.Errorf("publish_oncall providers failed: %w", errors.Join(slackErr, expressErr))
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
		if durable, ok := store.(OnCallDeliveryStore); ok {
			needs := false
			for kind, destination := range map[string]string{"slack": stringValue(team.SlackChannelID), "express": expressChatID} {
				if destination == "" {
					continue
				}
				version, err := publicationConfigVersion(ctx, store, team.ID, kind)
				if err != nil {
					return err
				}
				due, err := durable.NeedsOnCallPublication(ctx, db.PublicationAttempt{TeamID: team.ID, Provider: kind, Destination: destination, Fingerprint: fp, ConfigVersion: version}, now)
				if err != nil {
					return err
				}
				needs = needs || due
			}
			if !needs {
				continue
			}
		} else if team.OnCallAnnouncedUserIDs != nil && *team.OnCallAnnouncedUserIDs == fp {
			continue
		}
		_, durable := store.(OnCallDeliveryStore)
		if !durable && fp == "" && (team.OnCallAnnouncedUserIDs == nil || *team.OnCallAnnouncedUserIDs == "") {
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
		id := user.UserID.String()
		if !user.StartAt.IsZero() {
			id += ":" + user.StartAt.UTC().Format(time.RFC3339Nano) + ":" + user.EndAt.UTC().Format(time.RFC3339Nano)
		}
		ids = append(ids, id)
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

type OnCallDeliveryStore interface {
	RunOnCallDelivery(context.Context, db.PublicationAttempt, func() error) error
	NeedsOnCallPublication(context.Context, db.PublicationAttempt, time.Time) (bool, error)
}

func (p *PublishOnCallProcessor) sendOnCall(ctx context.Context, teamID uuid.UUID, kind, destination, fingerprint, key string, send func() error, generation ...*string) error {
	durable, ok := p.store.(OnCallDeliveryStore)
	if !ok {
		return send()
	}
	version, err := publicationConfigVersion(ctx, p.store, teamID, kind)
	if err != nil {
		return err
	}
	key += ":" + version
	if len(generation) > 0 {
		*generation[0] = key
	}
	attempt := db.PublicationAttempt{TeamID: teamID, Provider: kind, Destination: destination, Fingerprint: fingerprint, ConfigVersion: version, PublicationKey: key}
	if tracked, ok := p.store.(interface {
		RunOnCallDeliveryWithRef(context.Context, db.PublicationAttempt, uuid.UUID, func() (string, error)) error
	}); ok && kind == "express" && len(generation) > 1 {
		connector, err := p.store.GetIntegrationByKind(ctx, kind)
		if err != nil {
			return err
		}
		return tracked.RunOnCallDeliveryWithRef(ctx, attempt, connector.ID, func() (string, error) { err := send(); return *generation[1], err })
	}
	return durable.RunOnCallDelivery(ctx, attempt, send)
}
func publicationConfigVersion(ctx context.Context, store globalExpressIntegrationStore, teamID uuid.UUID, kind string) (string, error) {
	global, err := store.GetIntegrationByKind(ctx, kind)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	raw, _ := json.Marshal(global)
	if kind == "slack" {
		if scoped, ok := store.(interface {
			GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error)
			GetWorkspaceIntegration(context.Context, uuid.UUID, string) (db.Integration, error)
		}); ok {
			workspace, err := scoped.GetTeamWorkspaceID(ctx, teamID)
			if err != nil {
				return "", err
			}
			slot, err := scoped.GetWorkspaceIntegration(ctx, workspace, kind)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return "", err
			}
			slotRaw, _ := json.Marshal(slot)
			raw = append(raw, slotRaw...)
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}
