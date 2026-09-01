package documents

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Document é um artefato gerado (recibo, carteirinha, certificado...).
type Document struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	DocumentRef string `json:"document_ref"`
	QRToken     string `json:"qr_token"`
	Content     []byte `json:"content"`
	CreatedAt   string `json:"created_at"`
}

type Repo struct{}

// GetByToken retorna um documento resolvido pelo token do QR (escopo RLS do tenant).
func (r *Repo) GetByToken(ctx context.Context, tx pgx.Tx, token string) (*Document, error) {
	var d Document
	err := tx.QueryRow(ctx, `
		SELECT id::text, kind, title, document_ref, qr_token, content, created_at::text
		FROM documents WHERE qr_token = $1`, token).
		Scan(&d.ID, &d.Kind, &d.Title, &d.DocumentRef, &d.QRToken, &d.Content, &d.CreatedAt)
	return &d, err
}

// GetByID retorna um documento pelo ID (escopo RLS).
func (r *Repo) GetByID(ctx context.Context, tx pgx.Tx, id string) (*Document, error) {
	var d Document
	err := tx.QueryRow(ctx, `
		SELECT id::text, kind, title, document_ref, qr_token, content, created_at::text
		FROM documents WHERE id = $1::uuid`, id).
		Scan(&d.ID, &d.Kind, &d.Title, &d.DocumentRef, &d.QRToken, &d.Content, &d.CreatedAt)
	return &d, err
}

// TenantName devolve o nome do tenant do contexto (usado no cabeçalho do recibo).
func (r *Repo) TenantName(ctx context.Context, tx pgx.Tx) (string, error) {
	var name string
	err := tx.QueryRow(ctx, `SELECT name FROM tenants LIMIT 1`).Scan(&name)
	if err != nil {
		return "Chosen ERP", nil
	}
	return name, nil
}
