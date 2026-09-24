package documents

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// KindMembershipCard e o `kind` da carteirinha de membro. E o unico documento
// com pagina publica no webadmin (/member/{token}), entao quem monta link de
// envio precisa distingui-lo - ver delivery.PublicLink.
const KindMembershipCard = "membership_card"

// Document e um artefato gerado (recibo, carteirinha, certificado...).
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

// TenantName devolve o nome do tenant do contexto (usado no cabecalho do recibo).
func (r *Repo) TenantName(ctx context.Context, tx pgx.Tx) (string, error) {
	var name string
	err := tx.QueryRow(ctx, `SELECT name FROM tenants LIMIT 1`).Scan(&name)
	if err != nil {
		return "Chosen ERP", nil
	}
	return name, nil
}

// CardInfo resume os dados de uma carteirinha resolvida por token (app do membro).
type CardInfo struct {
	Token      string  `json:"token"`
	Ref        string  `json:"card_ref"`
	MemberID   string  `json:"member_id"`
	MemberName string  `json:"member"`
	TenantID   string  `json:"tenant_id"`
	BranchID   string  `json:"branch_id"`
	// PhotoURL e BranchName alimentam a carteirinha impressa; a foto vem do
	// cadastro do membro (members.photo_url) e a filial e o nome legivel, nao o
	// UUID - a versao anterior imprimia "Ref: CARD-XXXX" sem foto nem filial.
	PhotoURL   *string `json:"photo_url,omitempty"`
	BranchName *string `json:"branch_name,omitempty"`
}

// ResolveCard localiza a carteirinha de membro pelo token do QR. Deve ser
// chamado em escopo "Sede" (system) para resolver o token independente do RLS.
func (r *Repo) ResolveCard(ctx context.Context, tx pgx.Tx, token string) (*CardInfo, error) {
	var c CardInfo
	err := tx.QueryRow(ctx, `
		SELECT d.qr_token, d.document_ref, d.member_id::text, d.content->>'member',
		       d.tenant_id::text, COALESCE(m.branch_id::text, ''),
		       m.photo_url, b.name
		FROM documents d
		LEFT JOIN members m ON m.id = d.member_id
		LEFT JOIN branches b ON b.id = m.branch_id
		WHERE d.qr_token = $1 AND d.kind = 'membership_card'`, token).
		Scan(&c.Token, &c.Ref, &c.MemberID, &c.MemberName, &c.TenantID, &c.BranchID,
			&c.PhotoURL, &c.BranchName)
	return &c, err
}
