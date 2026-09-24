package documents

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Delivery e um envio de documento (outbox) por e-mail/WhatsApp.
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

// QueueDelivery registra um pedido de envio na outbox (entrega real e feita pelo worker).
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

// PendingDelivery e um item da outbox aguardando envio (usado pelo worker).
type PendingDelivery struct {
	ID         string
	TenantID   string
	BranchID   string
	DocumentID string
	Channel    string
	Recipient  string
}

// ListPending devolve os itens da outbox pendentes ou com falha elegivel a
// retry (attempts < maxAttempts). Deve ser chamado em escopo "Sede" (system)
// para enxergar todas as filiais.
func (r *Repo) ListPending(ctx context.Context, tx pgx.Tx, limit, maxAttempts int) ([]PendingDelivery, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, tenant_id::text, branch_id::text, document_id::text, channel, recipient
		FROM document_deliveries
		WHERE status IN ('pending','failed') AND attempts < $1
		ORDER BY created_at
		LIMIT $2`, maxAttempts, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PendingDelivery{}
	for rows.Next() {
		var p PendingDelivery
		if err := rows.Scan(&p.ID, &p.TenantID, &p.BranchID, &p.DocumentID, &p.Channel, &p.Recipient); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// MarkSent registra a entrega bem-sucedida de um item da outbox.
func (r *Repo) MarkSent(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `
		UPDATE document_deliveries
		SET status='sent', attempts = attempts + 1, sent_at=$2, error=NULL
		WHERE id = $1::uuid`, id, time.Now().UTC())
	return err
}

// MarkFailed registra a falha de entrega, tornando o item elegivel a retry.
func (r *Repo) MarkFailed(ctx context.Context, tx pgx.Tx, id, errMessage string) error {
	_, err := tx.Exec(ctx, `
		UPDATE document_deliveries
		SET status='failed', attempts = attempts + 1, error=$2
		WHERE id = $1::uuid`, id, errMessage)
	return err
}
