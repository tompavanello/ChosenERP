package governance

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- Convenios e documentacao legal (G6) ----

// LegalDocument e um documento legal com (opcional) vencimento.
type LegalDocument struct {
	ID          string  `json:"id"`
	BranchID    string  `json:"branch_id"`
	Kind        string  `json:"kind"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Reference   *string `json:"reference,omitempty"`
	IssuedAt    *string `json:"issued_at,omitempty"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
	FileURL     *string `json:"file_url,omitempty"`
	// Dias ate o vencimento (negativo = vencido). Ausente quando nao ha data.
	DaysToExpiry *int      `json:"days_to_expiry,omitempty"`
	Expired      bool      `json:"expired"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// LegalInput e o corpo de criacao/edicao de um documento legal.
type LegalInput struct {
	Kind        *string `json:"kind"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Reference   *string `json:"reference"`
	IssuedAt    *string `json:"issued_at"`
	ExpiresAt   *string `json:"expires_at"`
	FileURL     *string `json:"file_url"`
}

const legalCols = `id::text, branch_id::text, kind, title, description, reference,
	issued_at::text, expires_at::text, file_url,
	(CASE WHEN expires_at IS NULL THEN NULL ELSE (expires_at - current_date)::int END),
	(expires_at IS NOT NULL AND expires_at < current_date),
	created_at, updated_at`

func scanLegal(row pgx.Row) (*LegalDocument, error) {
	var d LegalDocument
	err := row.Scan(&d.ID, &d.BranchID, &d.Kind, &d.Title, &d.Description, &d.Reference,
		&d.IssuedAt, &d.ExpiresAt, &d.FileURL, &d.DaysToExpiry, &d.Expired, &d.CreatedAt, &d.UpdatedAt)
	return &d, err
}

func (r *Repo) ListLegalDocuments(ctx context.Context, tx pgx.Tx, kind string) ([]LegalDocument, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+legalCols+` FROM legal_documents
		WHERE ($1 = '' OR kind = $1)
		ORDER BY (expires_at IS NULL), expires_at, title LIMIT 500`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LegalDocument{}
	for rows.Next() {
		d, err := scanLegal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *Repo) GetLegalDocument(ctx context.Context, tx pgx.Tx, id string) (*LegalDocument, error) {
	return scanLegal(tx.QueryRow(ctx, `SELECT `+legalCols+` FROM legal_documents WHERE id = $1::uuid`, id))
}

func (r *Repo) CreateLegalDocument(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, in LegalInput) (*LegalDocument, error) {
	kind := "convenio"
	if in.Kind != nil && *in.Kind != "" {
		kind = *in.Kind
	}
	var newID string
	err := tx.QueryRow(ctx, `
		INSERT INTO legal_documents (tenant_id, branch_id, kind, title, description, reference, issued_at, expires_at, file_url, created_by)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5, $6, NULLIF($7,'')::date, NULLIF($8,'')::date, $9, $10::uuid)
		RETURNING id::text`,
		tenantID, branchID, kind, in.Title, in.Description, in.Reference,
		str(in.IssuedAt), str(in.ExpiresAt), in.FileURL, actorID).Scan(&newID)
	if err != nil {
		return nil, err
	}
	return r.GetLegalDocument(ctx, tx, newID)
}

func (r *Repo) UpdateLegalDocument(ctx context.Context, tx pgx.Tx, id string, in LegalInput) (*LegalDocument, error) {
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE legal_documents SET
			kind = COALESCE($2, kind),
			title = COALESCE(NULLIF($3,''), title),
			description = COALESCE($4, description),
			reference = COALESCE($5, reference),
			issued_at = CASE WHEN $6::boolean THEN NULLIF($7,'')::date ELSE issued_at END,
			expires_at = CASE WHEN $8::boolean THEN NULLIF($9,'')::date ELSE expires_at END,
			file_url = COALESCE($10, file_url),
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text`,
		id, in.Kind, in.Title, in.Description, in.Reference,
		in.IssuedAt != nil, str(in.IssuedAt), in.ExpiresAt != nil, str(in.ExpiresAt), in.FileURL).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	return r.GetLegalDocument(ctx, tx, updatedID)
}

func (r *Repo) DeleteLegalDocument(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM legal_documents WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ---- Painel de mandatos (G5) ----

// Mandate e um mandato vigente ou recem-encerrado de um membro.
type Mandate struct {
	ID           string  `json:"id"`
	MemberID     string  `json:"member_id"`
	MemberName   string  `json:"member_name"`
	CargoID      string  `json:"cargo_id"`
	CargoName    string  `json:"cargo_name"`
	CargoKind    string  `json:"cargo_kind"`
	StartedAt    *string `json:"started_at,omitempty"`
	EndsAt       *string `json:"ends_at,omitempty"`
	Status       string  `json:"status"`
	DaysToExpiry *int    `json:"days_to_expiry,omitempty"`
	Expiring     bool    `json:"expiring"`
}

// ListMandates devolve os mandatos com vencimento dentro da janela (ou vencidos),
// alem dos que nao tem data. Inclui ativos e encerrados recentemente para dar o
// historico de quem exerceu cada cargo.
func (r *Repo) ListMandates(ctx context.Context, tx pgx.Tx, windowDays int) ([]Mandate, error) {
	if windowDays <= 0 {
		windowDays = 60
	}
	rows, err := tx.Query(ctx, `
		SELECT mc.id::text, mc.member_id::text, m.full_name, mc.cargo_id::text, c.name, c.kind,
			mc.started_at::text, mc.ends_at::text, mc.status,
			(CASE WHEN mc.ends_at IS NULL THEN NULL ELSE (mc.ends_at - current_date)::int END) AS days,
			(mc.status = 'ativo' AND mc.ends_at IS NOT NULL AND (mc.ends_at - current_date) <= $1) AS expiring
		FROM member_cargos mc
		JOIN members m ON m.id = mc.member_id
		JOIN cargos c ON c.id = mc.cargo_id
		ORDER BY (mc.status <> 'ativo'), expiring DESC, mc.ends_at NULLS LAST, m.full_name`, windowDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Mandate{}
	for rows.Next() {
		var md Mandate
		if err := rows.Scan(&md.ID, &md.MemberID, &md.MemberName, &md.CargoID, &md.CargoName, &md.CargoKind,
			&md.StartedAt, &md.EndsAt, &md.Status, &md.DaysToExpiry, &md.Expiring); err != nil {
			return nil, err
		}
		out = append(out, md)
	}
	return out, rows.Err()
}
