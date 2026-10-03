package documents

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/jackc/pgx/v5"
)

// Kinds dos documentos de secretaria (certidoes e cartas).
const (
	KindCertificate = "certificate"
	KindLetter      = "letter"
)

// newToken gera um token aleatorio hexadecimal (QR do documento).
func newToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// IssueMemberDocument grava um documento (certificado/carta) do membro no mesmo
// `documents` usado por recibos/carteirinhas e devolve o registro com ref/token.
func (r *Repo) IssueMemberDocument(ctx context.Context, tx pgx.Tx, tenantID, branchID, memberID, kind, title, ref string, content []byte) (*Document, error) {
	token := newToken(16)
	var d Document
	err := tx.QueryRow(ctx, `
		INSERT INTO documents (tenant_id, branch_id, kind, title, document_ref, qr_token, content, member_id)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5, $6, $7, $8::uuid)
		RETURNING id::text, kind, title, document_ref, qr_token, created_at::text`,
		tenantID, branchID, kind, title, ref, token, content, memberID).
		Scan(&d.ID, &d.Kind, &d.Title, &d.DocumentRef, &d.QRToken, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListByMember lista os certificados/cartas emitidos para o membro.
func (r *Repo) ListByMember(ctx context.Context, tx pgx.Tx, memberID string) ([]Document, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, kind, title, document_ref, qr_token, created_at::text
		FROM documents
		WHERE member_id = $1::uuid AND kind IN ('certificate', 'letter')
		ORDER BY created_at DESC`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Document{}
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.Kind, &d.Title, &d.DocumentRef, &d.QRToken, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// NewRef gera uma referencia legivel para um documento (ex.: CERT-1A2B3C).
func NewRef(prefix string) string {
	return prefix + "-" + newToken(3)
}
