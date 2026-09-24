package suppliers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Supplier é um fornecedor (PF ou PJ). CPF e CNPJ são opcionais.
type Supplier struct {
	ID        string    `json:"id"`
	BranchID  *string   `json:"branch_id,omitempty"`
	Name      string    `json:"name"`
	TradeName *string   `json:"trade_name,omitempty"`
	CPF       *string   `json:"cpf,omitempty"`
	CNPJ      *string   `json:"cnpj,omitempty"`
	Email     *string   `json:"email,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateInput struct {
	Name      string  `json:"name"`
	TradeName *string `json:"trade_name"`
	CPF       *string `json:"cpf"`
	CNPJ      *string `json:"cnpj"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
	Notes     *string `json:"notes"`
}

type UpdateInput struct {
	Name      *string `json:"name"`
	TradeName *string `json:"trade_name"`
	CPF       *string `json:"cpf"`
	CNPJ      *string `json:"cnpj"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
	Notes     *string `json:"notes"`
	IsActive  *bool   `json:"is_active"`
}

type Repo struct{}

const supplierCols = `id::text, COALESCE(branch_id::text,''), name, trade_name, cpf, cnpj,
	email, phone, notes, is_active, created_at, updated_at`

func scan(row pgx.Row) (*Supplier, error) {
	var s Supplier
	var bid string
	if err := row.Scan(&s.ID, &bid, &s.Name, &s.TradeName, &s.CPF, &s.CNPJ,
		&s.Email, &s.Phone, &s.Notes, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	if bid != "" {
		s.BranchID = &bid
	}
	return &s, nil
}

// List retorna fornecedores do escopo (RLS), com busca opcional por nome,
// nome fantasia, CPF ou CNPJ.
func (r *Repo) List(ctx context.Context, tx pgx.Tx, q string) ([]Supplier, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+supplierCols+`
		FROM suppliers
		WHERE $1 = ''
		   OR name ILIKE '%' || $1 || '%'
		   OR COALESCE(trade_name,'') ILIKE '%' || $1 || '%'
		   OR COALESCE(cpf,'') ILIKE '%' || $1 || '%'
		   OR COALESCE(cnpj,'') ILIKE '%' || $1 || '%'
		ORDER BY name LIMIT 300`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Supplier{}
	for rows.Next() {
		s, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Supplier, error) {
	return scan(tx.QueryRow(ctx, `SELECT `+supplierCols+` FROM suppliers WHERE id = $1::uuid`, id))
}

func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateInput) (*Supplier, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO suppliers (tenant_id, branch_id, name, trade_name, cpf, cnpj, email, phone, notes)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text`,
		tenantID, branchID, in.Name, in.TradeName, in.CPF, in.CNPJ, in.Email, in.Phone, in.Notes).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, id)
}

// Update aplica semântica de PATCH: campo nil mantém o valor atual.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*Supplier, error) {
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE suppliers SET
			name = COALESCE($2, name),
			trade_name = COALESCE($3, trade_name),
			cpf = COALESCE($4, cpf),
			cnpj = COALESCE($5, cnpj),
			email = COALESCE($6, email),
			phone = COALESCE($7, phone),
			notes = COALESCE($8, notes),
			is_active = COALESCE($9::boolean, is_active),
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text`,
		id, in.Name, in.TradeName, in.CPF, in.CNPJ, in.Email, in.Phone, in.Notes, in.IsActive).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, updatedID)
}

func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM suppliers WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
