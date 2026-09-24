package org

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// AdminTenant e a visao administrativa de uma igreja (tenant) para a tela de
// onboarding do super_admin. Nao e escopada por RLS: vem de list_tenants().
type AdminTenant struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Plan        string    `json:"plan"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	BranchCount int       `json:"branch_count"`
	MemberCount int       `json:"member_count"`
}

// CreateTenant cria uma igreja com a base minima (Matriz, papeis e permissoes)
// via funcao SECURITY DEFINER. Devolve o id do tenant criado.
func (r *Repo) CreateTenant(ctx context.Context, tx pgx.Tx, name, slug, plan string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT create_tenant($1, $2, $3)`, name, slug, plan).Scan(&id)
	return id, err
}

// CreateTenantAdmin anexa o primeiro super_admin a igreja (identidade global).
func (r *Repo) CreateTenantAdmin(ctx context.Context, tx pgx.Tx, tenantID, email, passwordHash, fullName string) (string, error) {
	var userID string
	err := tx.QueryRow(ctx, `
		SELECT user_attach_to_tenant($1, $2, $3, $4::uuid, 'super_admin', NULL, true)`,
		email, passwordHash, fullName, tenantID).Scan(&userID)
	return userID, err
}

// ListTenants lista todas as igrejas (para o onboarding do super_admin).
func (r *Repo) ListTenants(ctx context.Context, tx pgx.Tx) ([]AdminTenant, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, name, slug, plan, is_active, created_at, branch_count, member_count
		FROM list_tenants()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdminTenant{}
	for rows.Next() {
		var t AdminTenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &t.IsActive,
			&t.CreatedAt, &t.BranchCount, &t.MemberCount); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
