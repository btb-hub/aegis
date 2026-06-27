package integrations

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type IncidentRef struct {
	ID           uuid.UUID
	TeamID       uuid.UUID
	AssigneeID   *uuid.UUID
	Status       string
	Severity     string
	Title        string
	Fingerprint  string
	JiraIssueKey *string
	CreatedAt    time.Time
}

type PageRecipient struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
	Locale      string
	SlackUserID *string
}

type TicketProvider interface {
	Kind() string
	CreateTicket(ctx context.Context, incident IncidentRef) (externalKey string, err error)
	TestConnection(ctx context.Context) error
}

type ChatProvider interface {
	Kind() string
	SendPage(ctx context.Context, incident IncidentRef, recipient PageRecipient) (messageRef string, err error)
	TestConnection(ctx context.Context) error
}

type IntegrationRow struct {
	ID      uuid.UUID
	Kind    string
	Name    string
	Config  []byte
	Enabled bool
}

type Loader interface {
	ListEnabledIntegrations(ctx context.Context) ([]IntegrationRow, error)
}

type Factory func(row IntegrationRow) (any, error)

type Registry struct {
	tickets map[string]TicketProvider
	chats   map[string]ChatProvider
}

func NewRegistry() *Registry {
	return &Registry{
		tickets: map[string]TicketProvider{},
		chats:   map[string]ChatProvider{},
	}
}

func Load(ctx context.Context, loader Loader, ticketFactories, chatFactories map[string]Factory) (*Registry, error) {
	rows, err := loader.ListEnabledIntegrations(ctx)
	if err != nil {
		return nil, err
	}

	reg := NewRegistry()
	for _, row := range rows {
		if factory, ok := ticketFactories[row.Kind]; ok {
			provider, err := factory(row)
			if err != nil {
				continue
			}
			if ticket, ok := provider.(TicketProvider); ok {
				reg.tickets[row.Kind] = ticket
			}
		}
		if factory, ok := chatFactories[row.Kind]; ok {
			provider, err := factory(row)
			if err != nil {
				continue
			}
			if chat, ok := provider.(ChatProvider); ok {
				reg.chats[row.Kind] = chat
			}
		}
	}
	return reg, nil
}

func (r *Registry) Ticket(kind string) (TicketProvider, bool) {
	p, ok := r.tickets[kind]
	return p, ok
}

func (r *Registry) Chat(kind string) (ChatProvider, bool) {
	p, ok := r.chats[kind]
	return p, ok
}

func (r *Registry) TicketProviders() []TicketProvider {
	out := make([]TicketProvider, 0, len(r.tickets))
	for _, p := range r.tickets {
		out = append(out, p)
	}
	return out
}

func (r *Registry) ChatProviders() []ChatProvider {
	out := make([]ChatProvider, 0, len(r.chats))
	for _, p := range r.chats {
		out = append(out, p)
	}
	return out
}

func (r *Registry) RegisterTicket(p TicketProvider) {
	r.tickets[p.Kind()] = p
}

func (r *Registry) RegisterChat(p ChatProvider) {
	r.chats[p.Kind()] = p
}

func ForEachTicket(reg *Registry, fn func(TicketProvider) error) {
	for _, provider := range reg.TicketProviders() {
		if err := fn(provider); err != nil {
			continue
		}
	}
}

func ForEachChat(reg *Registry, fn func(ChatProvider) error) {
	for _, provider := range reg.ChatProviders() {
		if err := fn(provider); err != nil {
			continue
		}
	}
}
