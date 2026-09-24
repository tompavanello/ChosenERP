package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"chosenerp/internal/store"
	"github.com/jackc/pgx/v5"
)

// errUnauthorized é o erro padrão de credenciais inválidas.
var errUnauthorized = fmt.Errorf("invalid credentials")

// Erros de MFA no login, distinguíveis pelo handler para orientar o frontend.
var (
	ErrMFARequired = errors.New("mfa_required")
	ErrMFAInvalid  = errors.New("mfa_invalid")
)

// ErrTenantForbidden sinaliza que a identidade não tem vínculo ativo com a
// igreja pedida (subdomínio/seletor).
var ErrTenantForbidden = errors.New("tenant_forbidden")

// selectionTTL é a validade do token de seleção de igreja (curto de propósito:
// só serve para completar o login).
const selectionTTL = 5 * time.Minute

// Membership é o vínculo de uma identidade com uma igreja (papel + filial).
type Membership struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
	Role       string `json:"role"`
	BranchID   string `json:"branch_id"`
	IsActive   bool   `json:"is_active"`
}

// TenantOption é uma igreja oferecida no seletor após o login.
type TenantOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Role string `json:"role"`
}

// LoginResult é o desfecho do login: ou entra direto (Profile+Tokens) ou exige
// que a identidade escolha a igreja (RequiresTenantSelection + SelectionToken).
type LoginResult struct {
	Profile                 *Profile
	Tokens                  *Tokens
	RequiresTenantSelection bool
	SelectionToken          string
	Tenants                 []TenantOption
}

// Profile representa o usuário autenticado e seu contexto de acesso.
type Profile struct {
	UserID       string
	Email        string
	FullName     string
	TenantID     string
	BranchID     string
	Role         string
	PasswordHash string
	Permissions  []string
	MFAEnabled   bool
	MFASecret    string
	// Memberships lista todas as igrejas da identidade (para o seletor).
	Memberships []Membership
}

// Tokens retorna o par access/refresh.
type Tokens struct {
	Access    string `json:"access_token"`
	Refresh   string `json:"refresh_token"`
	ExpiresIn int64  `json:"expires_in"`
}

// Service mantém a lógica de autenticação sobre o Store.
type Service struct {
	store      *store.Store
	secret     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewService(st *store.Store, secret string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{store: st, secret: secret, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// identityRow é a projeção da identidade (auth_lookup_user / auth_identity).
type identityRow struct {
	UserID       string
	FullName     string
	Email        string
	PasswordHash string
	MFAEnabled   bool
	MFASecret    string
	IsActive     bool
}

func scanIdentity(row pgx.Row) (*identityRow, error) {
	var id identityRow
	err := row.Scan(&id.UserID, &id.FullName, &id.Email, &id.PasswordHash,
		&id.MFAEnabled, &id.MFASecret, &id.IsActive)
	return &id, err
}

// findIdentityByEmail carrega a identidade via SECURITY DEFINER (fora do RLS),
// pois o tenant ainda não é conhecido na etapa de login.
func (s *Service) findIdentityByEmail(ctx context.Context, email string) (*identityRow, error) {
	id, err := scanIdentity(s.store.Pool().QueryRow(ctx, `SELECT * FROM auth_lookup_user($1)`, email))
	if err != nil {
		if store.IsNotFound(err) {
			return nil, errUnauthorized
		}
		return nil, err
	}
	return id, nil
}

func (s *Service) findIdentityByID(ctx context.Context, userID string) (*identityRow, error) {
	id, err := scanIdentity(s.store.Pool().QueryRow(ctx, `SELECT * FROM auth_identity($1::uuid)`, userID))
	if err != nil {
		if store.IsNotFound(err) {
			return nil, errUnauthorized
		}
		return nil, err
	}
	return id, nil
}

// Memberships lista as igrejas de uma identidade. Usa função SECURITY DEFINER:
// pode ser chamada na seleção de igreja (sem tenant) e no middleware.
func (s *Service) Memberships(ctx context.Context, userID string) ([]Membership, error) {
	rows, err := s.store.Pool().Query(ctx, `
		SELECT tenant_id, tenant_name, tenant_slug, role_key, COALESCE(branch_id,''), is_active
		FROM auth_memberships($1::uuid)`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Membership{}
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.TenantID, &m.TenantName, &m.TenantSlug, &m.Role, &m.BranchID, &m.IsActive); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func activeMemberships(ms []Membership) []Membership {
	out := []Membership{}
	for _, m := range ms {
		if m.IsActive {
			out = append(out, m)
		}
	}
	return out
}

func tenantOptions(ms []Membership) []TenantOption {
	out := []TenantOption{}
	for _, m := range ms {
		out = append(out, TenantOption{ID: m.TenantID, Name: m.TenantName, Slug: m.TenantSlug, Role: m.Role})
	}
	return out
}

func findMembership(ms []Membership, tenantID string) (Membership, bool) {
	for _, m := range ms {
		if m.IsActive && m.TenantID == tenantID {
			return m, true
		}
	}
	return Membership{}, false
}

// Login valida credenciais e resolve a igreja ativa.
//
//   - tenantSlug informado (subdomínio): exige membership ativa naquela igreja.
//   - sem slug: 1 membership entra direto; >1 exige seleção (token curto + lista).
func (s *Service) Login(ctx context.Context, email, password, code, tenantSlug string) (*LoginResult, error) {
	id, err := s.findIdentityByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if !CheckPassword(id.PasswordHash, password) {
		return nil, errUnauthorized
	}
	if !id.IsActive {
		return nil, errUnauthorized
	}
	// MFA: só exige o código depois da senha correta, para não revelar que a
	// conta tem MFA a quem não sabe a senha.
	if id.MFAEnabled {
		if strings.TrimSpace(code) == "" {
			return nil, ErrMFARequired
		}
		if !ValidateTOTP(id.MFASecret, code) {
			return nil, ErrMFAInvalid
		}
	}

	all, err := s.Memberships(ctx, id.UserID)
	if err != nil {
		return nil, err
	}
	active := activeMemberships(all)
	if len(active) == 0 {
		return nil, ErrTenantForbidden
	}

	if tenantSlug != "" {
		for _, m := range active {
			if m.TenantSlug == tenantSlug {
				return s.issueForMembership(ctx, id, all, m)
			}
		}
		return nil, ErrTenantForbidden
	}

	if len(active) == 1 {
		return s.issueForMembership(ctx, id, all, active[0])
	}

	token, err := NewToken(s.secret, Claims{UserID: id.UserID, Type: SelectionTokenType}, selectionTTL)
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		RequiresTenantSelection: true,
		SelectionToken:          token,
		Tenants:                 tenantOptions(active),
	}, nil
}

// issueForMembership monta o perfil e emite o par de tokens para a igreja escolhida.
func (s *Service) issueForMembership(ctx context.Context, id *identityRow, all []Membership, m Membership) (*LoginResult, error) {
	prof := &Profile{
		UserID:      id.UserID,
		Email:       id.Email,
		FullName:    id.FullName,
		TenantID:    m.TenantID,
		BranchID:    m.BranchID,
		Role:        m.Role,
		MFAEnabled:  id.MFAEnabled,
		Memberships: all,
	}
	err := s.store.WithTenant(ctx, store.Bounds{
		TenantID: m.TenantID, BranchID: m.BranchID, Role: m.Role, UserID: id.UserID,
	}, func(tx pgx.Tx) error {
		perms, err := s.rolePermissions(ctx, tx, m.Role, m.TenantID)
		if err != nil {
			return err
		}
		if perms == nil {
			perms = []string{}
		}
		prof.Permissions = perms
		_, _ = tx.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1::uuid`, id.UserID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	access, err := NewToken(s.secret, Claims{
		UserID: id.UserID, TenantID: m.TenantID, BranchID: m.BranchID,
		Role: m.Role, Type: AccessTokenType,
	}, s.accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := NewToken(s.secret, Claims{
		UserID: id.UserID, TenantID: m.TenantID, BranchID: m.BranchID,
		Role: m.Role, Type: RefreshTokenType,
	}, s.refreshTTL)
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		Profile: prof,
		Tokens:  &Tokens{Access: access, Refresh: refresh, ExpiresIn: int64(s.accessTTL.Seconds())},
	}, nil
}

// SelectTenant conclui o login escolhendo a igreja a partir do selection_token.
func (s *Service) SelectTenant(ctx context.Context, selectionToken, tenantID string) (*Profile, *Tokens, error) {
	claims, err := ParseToken(s.secret, selectionToken)
	if err != nil || claims.Type != SelectionTokenType {
		return nil, nil, ErrTenantForbidden
	}
	return s.switchTo(ctx, claims.UserID, tenantID)
}

// SwitchTenant troca a igreja ativa de uma sessão já autenticada.
func (s *Service) SwitchTenant(ctx context.Context, userID, tenantID string) (*Profile, *Tokens, error) {
	return s.switchTo(ctx, userID, tenantID)
}

func (s *Service) switchTo(ctx context.Context, userID, tenantID string) (*Profile, *Tokens, error) {
	id, err := s.findIdentityByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	all, err := s.Memberships(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	m, ok := findMembership(all, tenantID)
	if !ok {
		return nil, nil, ErrTenantForbidden
	}
	res, err := s.issueForMembership(ctx, id, all, m)
	if err != nil {
		return nil, nil, err
	}
	return res.Profile, res.Tokens, nil
}

// Me reconstrói o perfil completo (dados do usuário + permissões + igrejas) a
// partir das claims do JWT. É o que sustenta o GET /api/v1/me depois de um reload,
// quando já não existe o payload do login. Roda dentro do próprio escopo RLS.
func (s *Service) Me(ctx context.Context, userID, tenantID, branchID, role string) (*Profile, error) {
	prof := &Profile{
		UserID:   userID,
		TenantID: tenantID,
		BranchID: branchID,
		Role:     role,
	}
	err := s.store.WithTenant(ctx, store.Bounds{
		TenantID: tenantID, BranchID: branchID, Role: role, UserID: userID,
	}, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			SELECT u.id::text, u.email, u.full_name, u.mfa_enabled
			FROM users u
			WHERE u.id = $1::uuid`, userID).
			Scan(&prof.UserID, &prof.Email, &prof.FullName, &prof.MFAEnabled); err != nil {
			return err
		}
		perms, err := s.rolePermissions(ctx, tx, role, tenantID)
		if err != nil {
			return err
		}
		if perms == nil {
			perms = []string{}
		}
		prof.Permissions = perms
		return nil
	})
	if err != nil {
		return nil, err
	}
	// As memberships vêm de função SECURITY DEFINER (independem da RLS da sessão).
	if ms, err := s.Memberships(ctx, userID); err == nil {
		prof.Memberships = ms
	} else {
		prof.Memberships = []Membership{}
	}
	return prof, nil
}

// Refresh troca um refresh token válido por um novo par de tokens.
func (s *Service) Refresh(ctx context.Context, raw string) (*Tokens, error) {
	claims, err := ParseToken(s.secret, raw)
	if err != nil {
		return nil, err
	}
	if claims.Type != RefreshTokenType {
		return nil, fmt.Errorf("not a refresh token")
	}
	access, err := NewToken(s.secret, Claims{
		UserID: claims.UserID, TenantID: claims.TenantID, BranchID: claims.BranchID,
		Role: claims.Role, Type: AccessTokenType,
	}, s.accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := NewToken(s.secret, Claims{
		UserID: claims.UserID, TenantID: claims.TenantID, BranchID: claims.BranchID,
		Role: claims.Role, Type: RefreshTokenType,
	}, s.refreshTTL)
	if err != nil {
		return nil, err
	}
	return &Tokens{Access: access, Refresh: refresh, ExpiresIn: int64(s.accessTTL.Seconds())}, nil
}

func (s *Service) rolePermissions(ctx context.Context, tx pgx.Tx, roleKey, tenantID string) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT p.key
		FROM role_permissions rp
		JOIN roles r ON r.id = rp.role_id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE r.key = $1 AND r.tenant_id = $2`, roleKey, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// ---- MFA (TOTP) ----

// SetupMFA gera e grava um segredo TOTP (ainda não habilitado). O segredo só
// passa a valer no login depois de ConfirmMFA.
func (s *Service) SetupMFA(ctx context.Context, b store.Bounds, userID string) (string, error) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		return "", err
	}
	err = s.store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx,
			`UPDATE users SET mfa_secret = $2, mfa_enabled = false, updated_at = now() WHERE id = $1::uuid`,
			userID, secret)
		return e
	})
	if err != nil {
		return "", err
	}
	return secret, nil
}

// MFASecret devolve o segredo e o estado atual do MFA.
func (s *Service) MFASecret(ctx context.Context, b store.Bounds, userID string) (string, bool, error) {
	var secret string
	var enabled bool
	err := s.store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT COALESCE(mfa_secret,''), mfa_enabled FROM users WHERE id = $1::uuid`,
			userID).Scan(&secret, &enabled)
	})
	return secret, enabled, err
}

// EnableMFA confere o código do autenticador e liga o MFA.
func (s *Service) EnableMFA(ctx context.Context, b store.Bounds, userID, code string) error {
	secret, _, err := s.MFASecret(ctx, b, userID)
	if err != nil {
		return err
	}
	if secret == "" {
		return errors.New("mfa não iniciado")
	}
	if !ValidateTOTP(secret, code) {
		return ErrMFAInvalid
	}
	return s.store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx,
			`UPDATE users SET mfa_enabled = true, updated_at = now() WHERE id = $1::uuid`, userID)
		return e
	})
}

// DisableMFA desliga o MFA e apaga o segredo.
func (s *Service) DisableMFA(ctx context.Context, b store.Bounds, userID string) error {
	return s.store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx,
			`UPDATE users SET mfa_enabled = false, mfa_secret = NULL, updated_at = now() WHERE id = $1::uuid`, userID)
		return e
	})
}

// ---- Perfil do próprio usuário ----

// ErrWrongPassword sinaliza senha atual incorreta na troca de senha.
var ErrWrongPassword = errors.New("senha atual incorreta")

// UpdateProfile altera nome e e-mail do próprio usuário. O e-mail é único por
// tenant; a violação de unicidade é traduzida pelo handler.
func (s *Service) UpdateProfile(ctx context.Context, b store.Bounds, userID, fullName, email string) error {
	return s.store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		tag, e := tx.Exec(ctx, `
			UPDATE users SET
				full_name = COALESCE(NULLIF($2,''), full_name),
				email = COALESCE(NULLIF($3,''), email),
				updated_at = now()
			WHERE id = $1::uuid`, userID, fullName, email)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// ChangePassword confere a senha atual e grava a nova (bcrypt).
func (s *Service) ChangePassword(ctx context.Context, b store.Bounds, userID, current, newPassword string) error {
	return s.store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		var hash string
		if e := tx.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1::uuid`, userID).Scan(&hash); e != nil {
			return e
		}
		if !CheckPassword(hash, current) {
			return ErrWrongPassword
		}
		newHash, e := HashPassword(newPassword)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1::uuid`, userID, newHash)
		return e
	})
}
