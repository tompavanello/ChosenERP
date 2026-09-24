package announcements

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	StatusPending = "pending"
	StatusSent    = "sent"
	StatusFailed  = "failed"

	// Origens da fila de envio (mirror do CHECK da migração 000040).
	SourceManual         = "manual"
	SourceBirthday       = "birthday"
	SourceRoster         = "roster"
	SourceVisitorWelcome = "visitor_welcome"
)

type Delivery struct {
	ID                string  `json:"id"`
	AnnouncementID    string  `json:"announcement_id"`
	Channel           string  `json:"channel"`
	Recipient         string  `json:"recipient"`
	RecipientName     string  `json:"recipient_name,omitempty"`
	Source            string  `json:"source"`
	Provider          string  `json:"provider"`
	ProviderMessageID *string `json:"provider_message_id,omitempty"`
	Status            string  `json:"status"`
	Error             *string `json:"error,omitempty"`
	Attempts          int     `json:"attempts"`
	SentAt            *string `json:"sent_at,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

type Recipient struct {
	Phone      string
	FullName   string
	EntityType string
	EntityID   string
}

type SendInput struct {
	Channel        string         `json:"channel"`
	Audience       string         `json:"audience"`
	AudienceFilter AudienceFilter `json:"audience_filter,omitempty"`
}

type Stats struct {
	Pending int `json:"pending"`
	Sent    int `json:"sent"`
	Failed  int `json:"failed"`
	Total   int `json:"total"`
}

func (r *Repo) CreateDelivery(ctx context.Context, tx pgx.Tx, tenantID, branchID, announcementID, channel, provider, recipient string) (*Delivery, error) {
	var d Delivery
	err := tx.QueryRow(ctx, `
		INSERT INTO announcement_deliveries (tenant_id, branch_id, announcement_id, channel, provider, recipient)
		VALUES ($1, NULLIF($2,'')::uuid, $3::uuid, $4, $5, $6)
		RETURNING id::text, announcement_id::text, channel, recipient, provider, provider_message_id,
		          status, error, attempts, sent_at::text, created_at::text`,
		tenantID, branchID, announcementID, channel, provider, recipient).
		Scan(&d.ID, &d.AnnouncementID, &d.Channel, &d.Recipient, &d.Provider, &d.ProviderMessageID,
			&d.Status, &d.Error, &d.Attempts, &d.SentAt, &d.CreatedAt)
	return &d, err
}

// AutomatedDelivery é uma mensagem avulsa (sem announcement) enfileirada pelas
// automações (#31): aniversário, lembrete de escala e boas-vindas a visitante.
type AutomatedDelivery struct {
	TenantID      string
	BranchID      string
	Channel       string
	Provider      string
	Recipient     string
	RecipientName string
	Title         string
	Body          string
	Source        string
	DedupeKey     string
}

// CreateAutomatedDelivery enfileira uma mensagem avulsa. A chave de dedupe é
// única por tenant: quando já existe, devolve (false, nil) sem inserir — é o que
// impede o mesmo "feliz aniversário" de sair duas vezes no mesmo dia.
func (r *Repo) CreateAutomatedDelivery(ctx context.Context, tx pgx.Tx, in AutomatedDelivery) (bool, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO announcement_deliveries
			(tenant_id, branch_id, announcement_id, channel, provider, recipient,
			 recipient_name, title, body, source, dedupe_key)
		VALUES ($1, NULLIF($2,'')::uuid, NULL, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tenant_id, dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING
		RETURNING id::text`,
		in.TenantID, in.BranchID, in.Channel, in.Provider, in.Recipient,
		in.RecipientName, in.Title, in.Body, in.Source, in.DedupeKey).Scan(&id)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Repo) ListDeliveries(ctx context.Context, tx pgx.Tx, announcementID string, limit, offset int) ([]Delivery, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, COALESCE(announcement_id::text,''), channel, recipient,
		       COALESCE(recipient_name,''), source, provider, provider_message_id,
		       status, error, attempts, sent_at::text, created_at::text
		FROM announcement_deliveries
		WHERE announcement_id = $1::uuid
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, announcementID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Delivery{}
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.AnnouncementID, &d.Channel, &d.Recipient,
			&d.RecipientName, &d.Source, &d.Provider, &d.ProviderMessageID,
			&d.Status, &d.Error, &d.Attempts, &d.SentAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repo) Stats(ctx context.Context, tx pgx.Tx, announcementID string) (Stats, error) {
	var s Stats
	err := tx.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status='pending') AS pending,
			COUNT(*) FILTER (WHERE status='sent') AS sent,
			COUNT(*) FILTER (WHERE status='failed') AS failed,
			COUNT(*) AS total
		FROM announcement_deliveries WHERE announcement_id = $1::uuid`, announcementID).
		Scan(&s.Pending, &s.Sent, &s.Failed, &s.Total)
	return s, err
}

type PendingAnnouncementDelivery struct {
	ID             string
	TenantID       string
	BranchID       string
	AnnouncementID string
	Title          string
	Body           string
	Channel        string
	Recipient      string
	RecipientName  string
	Provider       string
	Source         string
}

// PendingDeliveries lista a fila pendente/falha. Mensagens avulsas (automações)
// não têm announcement_id associado, então o título/corpo vêm do snapshot
// gravado na própria linha; para os comunicados eles vêm de `announcements`.
func (r *Repo) PendingDeliveries(ctx context.Context, tx pgx.Tx, limit, maxAttempts int) ([]PendingAnnouncementDelivery, error) {
	rows, err := tx.Query(ctx, `
		SELECT ad.id::text, ad.tenant_id::text, COALESCE(ad.branch_id::text,''),
		       COALESCE(ad.announcement_id::text,''),
		       COALESCE(ad.title, a.title, ''), COALESCE(ad.body, a.body, ''),
		       ad.channel, ad.recipient, COALESCE(ad.recipient_name,''), ad.provider, ad.source
		FROM announcement_deliveries ad
		LEFT JOIN announcements a ON a.id = ad.announcement_id
		WHERE ad.status IN ('pending','failed') AND ad.attempts < $1
		ORDER BY ad.created_at
		LIMIT $2`, maxAttempts, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PendingAnnouncementDelivery{}
	for rows.Next() {
		var p PendingAnnouncementDelivery
		if err := rows.Scan(&p.ID, &p.TenantID, &p.BranchID,
			&p.AnnouncementID, &p.Title, &p.Body,
			&p.Channel, &p.Recipient, &p.RecipientName, &p.Provider, &p.Source); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repo) MarkSent(ctx context.Context, tx pgx.Tx, id, providerMessageID string) error {
	var pid *string
	if providerMessageID != "" {
		pid = &providerMessageID
	}
	_, err := tx.Exec(ctx, `
		UPDATE announcement_deliveries
		SET status='sent', attempts = attempts + 1, sent_at=$2,
		    provider_message_id = COALESCE($3, provider_message_id), error=NULL
		WHERE id = $1::uuid`, id, time.Now().UTC(), pid)
	return err
}

func (r *Repo) MarkFailed(ctx context.Context, tx pgx.Tx, id, errMessage string) error {
	_, err := tx.Exec(ctx, `
		UPDATE announcement_deliveries
		SET status='failed', attempts = attempts + 1, error=$2
		WHERE id = $1::uuid`, id, errMessage)
	return err
}
