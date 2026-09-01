package documents

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Delivery é um envio de documento (outbox) por e-mail/WhatsApp.
type Delivery struct {
	ID         string  `json:"id"`
	DocumentID string  `json:"document_id"`
	Channel    string  `json:"channel"`
	Recipient  string  `json:"recipient"`
	Status     string  `json:"status"`
	Error      *string `json:"error,omitempty"`
	Attempts   int     `json:"attempts"`
	SentAt     *string `json:"sent_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

// QueueDelivery registra um pedido de envio na outbox (entrega real é feita pelo worker).
func (r *Repo) QueueDelivery(ctx context.Context, tx pgx.Tx, tenantID, branchID, documentID, channel, recipient string) (*Delivery, error) {
	if channel != "email" && channel != "whatsapp" {
		return nil, errors.New("channel must be email or whatsapp")
	}
	if recipient == "" {
		return nil, errors.New("recipient required")
	}
	var d Delivery
	err := tx.QueryRow(ctx, `
		INSERT INTO document_deliveries (tenant_id, branch_id, document_id, channel, recipient)
		VALUES ($1, NULLIF($2,'')::uuid, $3::uuid, $4, $5)
		RETURNING id::text, document_id::text, channel, recipient, status, error, attempts,
		          sent_at::text, created_at::text`,
		tenantID, branchID, documentID, channel, recipient).
		Scan(&d.ID, &d.DocumentID, &d.Channel, &d.Recipient, &d.Status, &d.Error, &d.Attempts, &d.SentAt, &d.CreatedAt)
	return &d, err
}

func (r *Repo) ListDeliveries(ctx context.Context, tx pgx.Tx, documentID string) ([]Delivery, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, document_id::text, channel, recipient, status, error, attempts,
		       sent_at::text, created_at::text
		FROM document_deliveries WHERE document_id = $1::uuid ORDER BY created_at DESC`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Delivery{}
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.DocumentID, &d.Channel, &d.Recipient, &d.Status, &d.Error, &d.Attempts, &d.SentAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Dispatch simula o envio: em produção isto chama SMTP ou a API do WhatsApp.
// Aqui registramos apenas o resultado no próprio registro da outbox.
func (r *Repo) Dispatch(ctx context.Context, tx pgx.Tx, id string) (*Delivery, error) {
	now := time.Now().UTC()
	var d Delivery
	err := tx.QueryRow(ctx, `
		UPDATE document_deliveries
		SET status='sent', attempts = attempts + 1, sent_at=$2, error=NULL
		WHERE id = $1::uuid
		RETURNING id::text, document_id::text, channel, recipient, status, error, attempts,
		          sent_at::text, created_at::text`, id, now).
		Scan(&d.ID, &d.DocumentID, &d.Channel, &d.Recipient, &d.Status, &d.Error, &d.Attempts, &d.SentAt, &d.CreatedAt)
	return &d, err
}
