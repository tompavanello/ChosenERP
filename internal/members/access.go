package members

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Access descreve o acesso do membro ao app (identidade vinculada).
type Access struct {
	HasAccess          bool    `json:"has_access"`
	UserID             *string `json:"user_id,omitempty"`
	Email              *string `json:"email,omitempty"`
	Phone              *string `json:"phone,omitempty"`
	IsActive           *bool   `json:"is_active,omitempty"`
	MustChangePassword bool    `json:"must_change_password"`
	LastLoginAt        *string `json:"last_login_at,omitempty"`
}

var nonDigits = regexp.MustCompile(`\D+`)

// AccessInput e o corpo de criacao/edicao do acesso. Na edicao, campo nulo
// mantem; telefone vazio ("") limpa.
type AccessInput struct {
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Password string  `json:"password"`
	IsActive *bool   `json:"is_active"`
}

// GetAccess carrega a identidade vinculada ao membro via memberships.member_id.
// Sem vinculo devolve HasAccess=false (nao e erro).
func (r *Repo) GetAccess(ctx context.Context, tx pgx.Tx, memberID string) (*Access, error) {
	var a Access
	var userID, email, phone, lastLogin *string
	var isActive, mustChange bool
	err := tx.QueryRow(ctx, `
		SELECT u.id::text, u.email::text, u.phone, u.is_active,
		       u.last_login_at::text, u.must_change_password
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.member_id = $1::uuid`, memberID).
		Scan(&userID, &email, &phone, &isActive, &lastLogin, &mustChange)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &Access{HasAccess: false}, nil
		}
		return nil, err
	}
	a = Access{
		HasAccess: true, UserID: userID, Email: email, Phone: phone,
		IsActive: &isActive, LastLoginAt: lastLogin, MustChangePassword: mustChange,
	}
	return &a, nil
}

// SetAccess cria/anexa a identidade do membro com senha provisoria, via funcao
// SECURITY DEFINER user_attach_member (grava memberships.member_id). O hash da
// senha e calculado pelo chamador (o pacote members nao depende de auth).
func (r *Repo) SetAccess(ctx context.Context, tx pgx.Tx, tenantID, memberID, email, phone, passwordHash, fullName string) (*Access, error) {
	if _, err := tx.Exec(ctx, `
		SELECT user_attach_member($1, $2, $3, $4, $5::uuid, $6::uuid)`,
		strings.TrimSpace(email), normalizePhone(phone), passwordHash, fullName, tenantID, memberID); err != nil {
		return nil, err
	}
	return r.GetAccess(ctx, tx, memberID)
}

// UpdateAccess altera e-mail/telefone e ativa/desativa o acesso do membro.
// Semantica: campo ausente (nil) mantem; telefone vazio limpa.
func (r *Repo) UpdateAccess(ctx context.Context, tx pgx.Tx, memberID string, in AccessInput) (*Access, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE users u SET
			email = COALESCE(NULLIF($2,''), u.email),
			phone = CASE WHEN $3::boolean THEN NULLIF($4,'') ELSE u.phone END,
			is_active = COALESCE($5::boolean, u.is_active),
			updated_at = now()
		FROM memberships m
		WHERE m.user_id = u.id AND m.member_id = $1::uuid`,
		memberID, deref(in.Email), in.Phone != nil, normalizePhone(deref(in.Phone)), in.IsActive)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.GetAccess(ctx, tx, memberID)
}

// ResetAccessPassword define/redefine a senha do membro. `provisional` marca a
// senha como provisoria (forca troca no primeiro acesso). Recebe o hash pronto.
func (r *Repo) ResetAccessPassword(ctx context.Context, tx pgx.Tx, memberID, passwordHash string, provisional bool) error {
	tag, err := tx.Exec(ctx, `
		UPDATE users u SET
			password_hash = $2,
			must_change_password = $3,
			updated_at = now()
		FROM memberships m
		WHERE m.user_id = u.id AND m.member_id = $1::uuid`,
		memberID, passwordHash, provisional)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func normalizePhone(s string) string {
	return nonDigits.ReplaceAllString(s, "")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
