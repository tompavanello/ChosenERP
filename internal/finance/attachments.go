package finance

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Attachment é um documento comprobatório anexado a um lançamento
// (foto de recibo, nota fiscal, comprovante bancário, etc.).
type Attachment struct {
	ID            string    `json:"id"`
	TransactionID string    `json:"transaction_id"`
	FileName      string    `json:"file_name"`
	FileURL       string    `json:"file_url"`
	ContentType   *string   `json:"content_type,omitempty"`
	FileSize      int64     `json:"file_size"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateAttachmentInput é o payload para anexar um documento a um lançamento.
type CreateAttachmentInput struct {
	FileName    string  `json:"file_name"`
	FileURL     string  `json:"file_url"`
	ContentType *string `json:"content_type,omitempty"`
	FileSize    int64   `json:"file_size"`
}

// ListAttachments retorna os anexos de um lançamento no escopo atual.
func (r *Repo) ListAttachments(ctx context.Context, tx pgx.Tx, transactionID string) ([]Attachment, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, transaction_id::text, file_name, file_url, content_type, file_size, created_at
		FROM financial_attachments
		WHERE transaction_id = $1::uuid
		ORDER BY created_at DESC`, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Attachment{}
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.TransactionID, &a.FileName, &a.FileURL,
			&a.ContentType, &a.FileSize, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateAttachment registra um anexo para um lançamento no escopo.
func (r *Repo) CreateAttachment(ctx context.Context, tx pgx.Tx, tenantID, branchID, transactionID string, in CreateAttachmentInput) (*Attachment, error) {
	var a Attachment
	err := tx.QueryRow(ctx, `
		INSERT INTO financial_attachments
			(tenant_id, branch_id, transaction_id, file_name, file_url, content_type, file_size)
		VALUES ($1, NULLIF($2,'')::uuid, $3::uuid, $4, $5, $6, $7)
		RETURNING id::text, transaction_id::text, file_name, file_url, content_type, file_size, created_at`,
		tenantID, branchID, transactionID, in.FileName, in.FileURL, in.ContentType, in.FileSize).
		Scan(&a.ID, &a.TransactionID, &a.FileName, &a.FileURL, &a.ContentType, &a.FileSize, &a.CreatedAt)
	return &a, err
}

// GetAttachment retorna um anexo pelo id (para download/view).
func (r *Repo) GetAttachment(ctx context.Context, tx pgx.Tx, id string) (*Attachment, error) {
	var a Attachment
	err := tx.QueryRow(ctx, `
		SELECT id::text, transaction_id::text, file_name, file_url, content_type, file_size, created_at
		FROM financial_attachments WHERE id = $1::uuid`, id).
		Scan(&a.ID, &a.TransactionID, &a.FileName, &a.FileURL, &a.ContentType, &a.FileSize, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
