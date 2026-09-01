package auth

import (
	"context"
	"fmt"
	"time"

	"chosenerp/internal/store"
	"github.com/jackc/pgx/v5"
)

// errUnauthorized é o erro padrão de credenciais inválidas.
var errUnauthorized = fmt.Errorf("invalid credentials")

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

// Login valida credenciais e emite tokens, resolvendo tenant/branch/role.
// A consulta do usuário roda via função SECURITY DEFINER (fora do scopo RLS),
// pois o tenant/branch ainda não é conhecido nesta etapa.
func (s *Service) Login(ctx context.Context, email, password string) (*Profile, *Tokens, error) {
	var prof Profile
	err := s.store.Pool().QueryRow(ctx, `SELECT * FROM auth_lookup_user($1)`, email).
		Scan(&prof.UserID, &prof.FullName, &prof.Email, &prof.TenantID, &prof.BranchID, &prof.Role, &prof.PasswordHash)
	if err != nil {
		if store.IsNotFound(err) {
			return nil, nil, errUnauthorized
		}
		return nil, nil, err
	}
	if !CheckPassword(prof.PasswordHash, password) {
		return nil, nil, errUnauthorized
	}
	// Permissões dentro do contexto RLS resolvido.
	err = s.store.WithTenant(ctx, store.Bounds{TenantID: prof.TenantID, BranchID: prof.BranchID, Role: prof.Role},
		func(tx pgx.Tx) error {
			perms, err := s.rolePermissions(ctx, tx, prof.Role, prof.TenantID)
			if err != nil {
				return err
			}
			prof.Permissions = perms
			return nil
		})
	if err != nil {
		return nil, nil, err
	}
	access, err := NewToken(s.secret, Claims{
		UserID: prof.UserID, TenantID: prof.TenantID, BranchID: prof.BranchID,
		Role: prof.Role, Type: AccessTokenType,
	}, s.accessTTL)
	if err != nil {
		return nil, nil, err
	}
	refresh, err := NewToken(s.secret, Claims{
		UserID: prof.UserID, TenantID: prof.TenantID, BranchID: prof.BranchID,
		Role: prof.Role, Type: RefreshTokenType,
	}, s.refreshTTL)
	if err != nil {
		return nil, nil, err
	}
	return &prof, &Tokens{Access: access, Refresh: refresh, ExpiresIn: int64(s.accessTTL.Seconds())}, nil
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
