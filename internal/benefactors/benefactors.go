package benefactors

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Benefactor é um benfeitor externo (sem vínculo de membresia).
type Benefactor struct {
	ID        string    `json:"id"`
	BranchID  *string   `json:"branch_id,omitempty"`
	Name      string    `json:"name"`
	Email     *string   `json:"email,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateInput struct {
	Name  string  `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
	Notes *string `json:"notes"`
}

type Repo struct{}

func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]Benefactor, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, COALESCE(branch_id::text,''), name, email, phone, notes, created_at
		FROM benefactors ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Benefactor{}
	for rows.Next() {
		var b Benefactor
		var bid string
		if err := rows.Scan(&b.ID, &bid, &b.Name, &b.Email, &b.Phone, &b.Notes, &b.CreatedAt); err != nil {
			return nil, err
		}
		if bid != "" {
			b.BranchID = &bid
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateInput) (*Benefactor, error) {
	var b Benefactor
	err := tx.QueryRow(ctx, `
		INSERT INTO benefactors (tenant_id, branch_id, name, email, phone, notes)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5, $6)
		RETURNING id::text, COALESCE(branch_id::text,''), name, email, phone, notes, created_at`,
		tenantID, branchID, in.Name, in.Email, in.Phone, in.Notes).
		Scan(&b.ID, &b.BranchID, &b.Name, &b.Email, &b.Phone, &b.Notes, &b.CreatedAt)
	return &b, err
}
