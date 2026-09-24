// Package users gerencia os usuarios administrativos e o catalogo de perfis
// (RBAC) do tenant. O isolamento e do RLS; a autorizacao (quem pode gerir) e
// aplicada no handler.
package users

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/auth"
)

type User struct {
	ID          string    `json:"id"`
	BranchID    *string   `json:"branch_id,omitempty"`
	Role        string    `json:"role"`
	Email       string    `json:"email"`
	FullName    string    `json:"full_name"`
	IsActive    bool      `json:"is_active"`
	MFAEnabled  bool      `json:"mfa_enabled"`
	LastLoginAt *string   `json:"last_login_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Role struct {
	ID          string   `json:"id"`
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	IsSystem    bool     `json:"is_system"`
	Permissions []string `json:"permissions"`
	UserCount   int      `json:"user_count"`
}

type Permission struct {
	Key    string `json:"key"`
	Module string `json:"module"`
	Name   string `json:"name"`
}

type CreateInput struct {
	Email    string  `json:"email"`
	FullName string  `json:"full_name"`
	Password string  `json:"password"`
	RoleKey  string  `json:"role"`
	BranchID *string `json:"branch_id"`
	IsActive *bool   `json:"is_active"`
}

type UpdateInput struct {
	FullName *string `json:"full_name"`
	RoleKey  *string `json:"role"`
	BranchID *string `json:"branch_id"`
	IsActive *bool   `json:"is_active"`
}

type Repo struct{}

// userCols projeta a identidade junto com o VINCULO na igreja do contexto
// (papel + filial). O mesmo usuario pode ter vinculos diferentes por tenant.
const userCols = `u.id::text, m.branch_id::text, r.key, u.email::text, u.full_name,
	u.is_active, u.mfa_enabled, u.last_login_at::text, u.created_at`

const userFrom = `FROM users u
	JOIN memberships m ON m.user_id = u.id AND m.tenant_id = current_tenant()
	JOIN roles r ON r.id = m.role_id`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.BranchID, &u.Role, &u.Email, &u.FullName,
		&u.IsActive, &u.MFAEnabled, &u.LastLoginAt, &u.CreatedAt)
	return &u, err
}

func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*User, error) {
	return scanUser(tx.QueryRow(ctx, `
		SELECT `+userCols+`
		`+userFrom+`
		WHERE u.id = $1::uuid`, id))
}

func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]User, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+userCols+`
		`+userFrom+`
		ORDER BY u.full_name`)
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

// Create cria (ou anexa) a identidade e o vinculo com a igreja do contexto via
// funcao SECURITY DEFINER. Se o e-mail ja existe, NAO altera a senha existente -
// apenas cria o membership. Perfil inexistente => SQLSTATE P0002; vinculo
// duplicado => 23505 (o handler traduz).
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID string, in CreateInput) (*User, error) {
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
	return r.Get(ctx, tx, newID)
}

// Update separa identidade (users) de vinculo (memberships). `branch_id` tem
// semantica de tres estados: ausente mantem; presente e vazio limpa (Sede);
// presente e preenchido troca.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*User, error) {
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE users u SET
			full_name = COALESCE($2, u.full_name),
			is_active = COALESCE($3::boolean, u.is_active),
			updated_at = now()
		WHERE u.id = $1::uuid
		RETURNING u.id::text`,
		id, in.FullName, in.IsActive).
		Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE memberships m SET
			role_id = COALESCE((SELECT rr.id FROM roles rr
			                     WHERE rr.tenant_id = m.tenant_id AND rr.key = $2), m.role_id),
			branch_id = CASE WHEN $3::boolean THEN NULLIF($4,'')::uuid ELSE m.branch_id END,
			updated_at = now()
		WHERE m.user_id = $1::uuid AND m.tenant_id = current_tenant()`,
		id, in.RoleKey, in.BranchID != nil, strOrEmpty(in.BranchID)); err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, updatedID)
}

func (r *Repo) ResetPassword(ctx context.Context, tx pgx.Tx, id, password string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1::uuid`, id, hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListRoles devolve os perfis do tenant com suas permissoes e contagem de usuarios.
func (r *Repo) ListRoles(ctx context.Context, tx pgx.Tx) ([]Role, error) {
	rows, err := tx.Query(ctx, `
		SELECT r.id::text, r.key, r.name, r.is_system,
		       (SELECT count(*) FROM memberships m
		         WHERE m.role_id = r.id AND m.tenant_id = current_tenant())::int
		FROM roles r
		ORDER BY r.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Role{}
	idx := map[string]int{}
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Key, &role.Name, &role.IsSystem, &role.UserCount); err != nil {
			return nil, err
		}
		role.Permissions = []string{}
		idx[role.Key] = len(out)
		out = append(out, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	prows, err := tx.Query(ctx, `
		SELECT r.key, p.key
		FROM role_permissions rp
		JOIN roles r ON r.id = rp.role_id
		JOIN permissions p ON p.id = rp.permission_id
		ORDER BY r.key, p.key`)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	for prows.Next() {
		var roleKey, permKey string
		if err := prows.Scan(&roleKey, &permKey); err != nil {
			return nil, err
		}
		if i, ok := idx[roleKey]; ok {
			out[i].Permissions = append(out[i].Permissions, permKey)
		}
	}
	return out, prows.Err()
}

// ListPermissions devolve o catalogo global de permissoes.
func (r *Repo) ListPermissions(ctx context.Context, tx pgx.Tx) ([]Permission, error) {
	rows, err := tx.Query(ctx, `SELECT key, module, name FROM permissions ORDER BY module, key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Permission{}
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Key, &p.Module, &p.Name); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
