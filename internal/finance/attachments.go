package finance

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Attachment e um documento comprobatorio anexado a um lancamento ou a um
// agrupamento bancario (foto de recibo, nota fiscal, comprovante etc.).
// Exatamente um entre TransactionID e GroupID e preenchido.
type Attachment struct {
	ID            string    `json:"id"`
	TransactionID *string   `json:"transaction_id,omitempty"`
	GroupID       *string   `json:"group_id,omitempty"`
	FileName      string    `json:"file_name"`
	FileURL       string    `json:"file_url"`
	ContentType   *string   `json:"content_type,omitempty"`
	FileSize      int64     `json:"file_size"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateAttachmentInput e o payload para anexar um documento.
type CreateAttachmentInput struct {
	FileName    string  `json:"file_name"`
	FileURL     string  `json:"file_url"`
	ContentType *string `json:"content_type,omitempty"`
	FileSize    int64   `json:"file_size"`
}

const attachmentCols = `id::text, transaction_id::text, group_id::text,
	file_name, file_url, content_type, file_size, created_at`

func scanAttachment(row pgx.Row) (*Attachment, error) {
	var a Attachment
	if err := row.Scan(&a.ID, &a.TransactionID, &a.GroupID, &a.FileName, &a.FileURL,
		&a.ContentType, &a.FileSize, &a.CreatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAttachments retorna os anexos de um lancamento no escopo atual.
func (r *Repo) ListAttachments(ctx context.Context, tx pgx.Tx, transactionID string) ([]Attachment, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+attachmentCols+`
		FROM financial_attachments
		WHERE transaction_id = $1::uuid
		ORDER BY created_at DESC`, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Attachment{}
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// ListGroupAttachments retorna os anexos vinculados diretamente ao grupo e os
// anexos dos lancamentos que o compoem. E a lista usada pela consulta do
// agrupamento no financeiro.
func (r *Repo) ListGroupAttachments(ctx context.Context, tx pgx.Tx, groupID string) ([]Attachment, error) {
	rows, err := tx.Query(ctx, `
		SELECT fa.id::text, fa.transaction_id::text, fa.group_id::text,
		       fa.file_name, fa.file_url, fa.content_type, fa.file_size, fa.created_at
		FROM financial_attachments fa
		LEFT JOIN financial_transactions t ON t.id = fa.transaction_id
		WHERE fa.group_id = $1::uuid
		   OR (t.group_id = $1::uuid AND t.voided_at IS NULL)
		ORDER BY fa.created_at DESC`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Attachment{}
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// CreateAttachment registra um anexo para um lancamento no escopo.
func (r *Repo) CreateAttachment(ctx context.Context, tx pgx.Tx, tenantID, branchID, transactionID string, in CreateAttachmentInput) (*Attachment, error) {
	return scanAttachment(tx.QueryRow(ctx, `
		INSERT INTO financial_attachments
			(tenant_id, branch_id, transaction_id, file_name, file_url, content_type, file_size)
		VALUES ($1, NULLIF($2,'')::uuid, $3::uuid, $4, $5, $6, $7)
		RETURNING `+attachmentCols,
		tenantID, branchID, transactionID, in.FileName, in.FileURL, in.ContentType, in.FileSize))
}

// CreateGroupAttachment registra um anexo vinculado diretamente ao grupo.
func (r *Repo) CreateGroupAttachment(ctx context.Context, tx pgx.Tx, tenantID, branchID, groupID string, in CreateAttachmentInput) (*Attachment, error) {
	return scanAttachment(tx.QueryRow(ctx, `
		INSERT INTO financial_attachments
			(tenant_id, branch_id, group_id, file_name, file_url, content_type, file_size)
		VALUES ($1, NULLIF($2,'')::uuid, $3::uuid, $4, $5, $6, $7)
		RETURNING `+attachmentCols,
		tenantID, branchID, groupID, in.FileName, in.FileURL, in.ContentType, in.FileSize))
}

// GetAttachment retorna um anexo pelo id (para download/view).
func (r *Repo) GetAttachment(ctx context.Context, tx pgx.Tx, id string) (*Attachment, error) {
	return scanAttachment(tx.QueryRow(ctx, `
		SELECT `+attachmentCols+`
		FROM financial_attachments WHERE id = $1::uuid`, id))
}
