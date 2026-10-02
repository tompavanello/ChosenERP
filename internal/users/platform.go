package users

import (
	"context"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/auth"
)

// As funcoes abaixo atendem o CONSOLE DA PLATAFORMA: operam uma igreja explicita
// (por id), fora do escopo RLS do tenant. Sao usadas para suporte (reset de
// senha, definicao de administradores), sempre com autorizacao de platform admin
// no handler e registro em audit_log.

const platformUserCols = `u.id::text, m.branch_id::text, ro.key, u.email::text, u.full_name,
	u.is_active, u.mfa_enabled, u.last_login_at::text, u.created_at`

// ListByTenant lista os acessos vinculados a uma igreja.
func (r *Repo) ListByTenant(ctx context.Context, tx pgx.Tx, tenantID string) ([]User, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+platformUserCols+`
		FROM users u
		JOIN memberships m ON m.user_id = u.id AND m.tenant_id = $1::uuid
		JOIN roles ro ON ro.id = m.role_id
		ORDER BY u.full_name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

// GetByTenant carrega um acesso da igreja informada.
func (r *Repo) GetByTenant(ctx context.Context, tx pgx.Tx, tenantID, userID string) (*User, error) {
	return scanUser(tx.QueryRow(ctx, `
		SELECT `+platformUserCols+`
		FROM users u
		JOIN memberships m ON m.user_id = u.id AND m.tenant_id = $1::uuid
		JOIN roles ro ON ro.id = m.role_id
		WHERE u.id = $2::uuid`, tenantID, userID))
}

// CreateInTenant cria/anexa a identidade e o vinculo na igreja informada.
func (r *Repo) CreateInTenant(ctx context.Context, tx pgx.Tx, tenantID string, in CreateInput) (*User, error) {
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	var newID string
	err = tx.QueryRow(ctx, `
		SELECT user_attach_to_tenant($1, $2, $3, $4::uuid, $5, NULLIF($6,'')::uuid, $7)`,
		in.Email, hash, in.FullName, tenantID, in.RoleKey, strOrEmpty(in.BranchID), in.IsActive).
		Scan(&newID)
	if err != nil {
		return nil, err
	}
	return r.GetByTenant(ctx, tx, tenantID, newID)
}

// UpdateInTenant edita nome/ativo e o vinculo (perfil/filial) da igreja.
func (r *Repo) UpdateInTenant(ctx context.Context, tx pgx.Tx, tenantID, userID string, in UpdateInput) (*User, error) {
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE users u SET
			full_name = COALESCE($3, u.full_name),
			is_active = COALESCE($4::boolean, u.is_active),
			updated_at = now()
		WHERE u.id = $2::uuid
		  AND EXISTS (SELECT 1 FROM memberships m WHERE m.user_id = u.id AND m.tenant_id = $1::uuid)
		RETURNING u.id::text`,
		tenantID, userID, in.FullName, in.IsActive).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE memberships m SET
			role_id = COALESCE((SELECT rr.id FROM roles rr
			                     WHERE rr.tenant_id = $1::uuid AND rr.key = $3), m.role_id),
			branch_id = CASE WHEN $4::boolean THEN NULLIF($5,'')::uuid ELSE m.branch_id END,
			updated_at = now()
		WHERE m.user_id = $2::uuid AND m.tenant_id = $1::uuid`,
		tenantID, userID, in.RoleKey, in.RoleKey != nil, strOrEmpty(in.BranchID)); err != nil {
		return nil, err
	}
	return r.GetByTenant(ctx, tx, tenantID, updatedID)
}

// ResetPasswordInTenant redefine a senha de um acesso da igreja informada.
func (r *Repo) ResetPasswordInTenant(ctx context.Context, tx pgx.Tx, tenantID, userID, password string) error {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM memberships WHERE user_id = $2::uuid AND tenant_id = $1::uuid)`,
		tenantID, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	return r.ResetPassword(ctx, tx, userID, password)
}

// ListRolesByTenant devolve os perfis da igreja (para o seletor do console).
func (r *Repo) ListRolesByTenant(ctx context.Context, tx pgx.Tx, tenantID string) ([]Role, error) {
	rows, err := tx.Query(ctx, `
		SELECT r.id::text, r.key, r.name, r.is_system,
		       (SELECT count(*) FROM memberships m
		         WHERE m.role_id = r.id AND m.tenant_id = $1::uuid)::int
		FROM roles r
		WHERE r.tenant_id = $1::uuid
		ORDER BY r.name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Role{}
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Key, &role.Name, &role.IsSystem, &role.UserCount); err != nil {
			return nil, err
		}
		role.Permissions = []string{}
		out = append(out, role)
	}
	return out, rows.Err()
}
