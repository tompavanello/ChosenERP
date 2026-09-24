package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/auth"
	"chosenerp/internal/members"
	"chosenerp/internal/store"
)

// Config é a fatia da config global usada pelos handlers HTTP.
type Config struct {
	JWTSecret  string
	AppBaseURL string
	UploadDir  string
	// MemberPhotoMaxBytes limita o upload da foto do membro (0 = sem limite).
	MemberPhotoMaxBytes int64
}

// boundsFromClaims converte claims em Bounds para RLS.
func boundsFromClaims(c *auth.Claims) store.Bounds {
	return store.Bounds{TenantID: c.TenantID, BranchID: c.BranchID, Role: c.Role, UserID: c.UserID}
}

// profilePayload padroniza a resposta de perfil (login, /me e edição de perfil).
func profilePayload(p *auth.Profile) map[string]any {
	ms := p.Memberships
	if ms == nil {
		ms = []auth.Membership{}
	}
	perms := p.Permissions
	if perms == nil {
		perms = []string{}
	}
	return map[string]any{
		"id": p.UserID, "user_id": p.UserID, "email": p.Email, "full_name": p.FullName,
		"tenant_id": p.TenantID, "branch_id": p.BranchID, "role": p.Role,
		"permissions": perms, "mfa_enabled": p.MFAEnabled, "memberships": ms,
	}
}

// writeBranchID devolve a filial para gravação dentro de uma transação. Usuário
// de filial usa a própria; a Sede (branch vazio) cai na filial raiz do tenant
// (Sede Matriz), porque várias tabelas têm branch_id NOT NULL. O escopo RLS
// continua sendo o da sessão (Sede), e rls_write já admite is_headquarters().
func (a *App) writeBranchID(ctx context.Context, tx pgx.Tx, c *auth.Claims) (string, error) {
	if c.BranchID != "" {
		return c.BranchID, nil
	}
	id, err := store.DefaultBranchID(ctx, tx, c.TenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("nenhuma filial cadastrada: crie uma filial antes de lançar")
	}
	return id, err
}

// handleLogin autentica e retorna tokens — ou pede a escolha da igreja.
//
// `tenant_slug` é enviado pelo subdomínio (igreja.dominio) ou pelo domínio
// central, para já entrar na igreja certa.
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		Code       string `json:"code"` // código TOTP quando o usuário tem MFA
		TenantSlug string `json:"tenant_slug"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	res, err := a.Auth.Login(r.Context(), in.Email, in.Password, in.Code, in.TenantSlug)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrMFARequired):
			writeErr(w, http.StatusUnauthorized, "mfa_required")
		case errors.Is(err, auth.ErrMFAInvalid):
			writeErr(w, http.StatusUnauthorized, "mfa_invalid")
		case errors.Is(err, auth.ErrTenantForbidden):
			writeErr(w, http.StatusForbidden, "tenant_forbidden")
		default:
			writeErr(w, http.StatusUnauthorized, "invalid credentials")
		}
		return
	}
	if res.RequiresTenantSelection {
		writeJSON(w, http.StatusOK, map[string]any{
			"requires_tenant_selection": true,
			"selection_token":           res.SelectionToken,
			"tenants":                   res.Tenants,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tokens": res.Tokens,
		"user":   profilePayload(res.Profile),
	})
}

// handleSelectTenant conclui o login quando a identidade tem mais de uma igreja.
func (a *App) handleSelectTenant(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SelectionToken string `json:"selection_token"`
		TenantID       string `json:"tenant_id"`
	}
	if err := readJSON(r, &in); err != nil || in.SelectionToken == "" || in.TenantID == "" {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	prof, tokens, err := a.Auth.SelectTenant(r.Context(), in.SelectionToken, in.TenantID)
	if err != nil {
		if errors.Is(err, auth.ErrTenantForbidden) {
			writeErr(w, http.StatusForbidden, "tenant_forbidden")
			return
		}
		writeErr(w, http.StatusUnauthorized, "invalid selection token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": tokens, "user": profilePayload(prof)})
}

// handleSwitchTenant troca a igreja ativa de uma sessão autenticada.
func (a *App) handleSwitchTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		TenantID string `json:"tenant_id"`
	}
	if err := readJSON(r, &in); err != nil || in.TenantID == "" {
		writeErr(w, http.StatusBadRequest, "tenant_id required")
		return
	}
	prof, tokens, err := a.Auth.SwitchTenant(r.Context(), claims.UserID, in.TenantID)
	if err != nil {
		if errors.Is(err, auth.ErrTenantForbidden) {
			writeErr(w, http.StatusForbidden, "tenant_forbidden")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": tokens, "user": profilePayload(prof)})
}

// handleMeTenants lista as igrejas da identidade (para o seletor).
func (a *App) handleMeTenants(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	ms, err := a.Auth.Memberships(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tenants": ms})
}

// handleRefresh emite novo par de tokens a partir do refresh token.
func (a *App) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Refresh string `json:"refresh_token"`
	}
	if err := readJSON(r, &in); err != nil || in.Refresh == "" {
		writeErr(w, http.StatusBadRequest, "refresh_token required")
		return
	}
	tokens, err := a.Auth.Refresh(r.Context(), in.Refresh)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": tokens})
}

// handleMe retorna o perfil e permissões do usuário autenticado.
func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	prof, err := a.Auth.Me(r.Context(), claims.UserID, claims.TenantID, claims.BranchID, claims.Role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "profile unavailable")
		return
	}
	// Mesma forma do payload de login: o frontend usa este GET para reidratar a
	// sessão depois de um reload, e depende de `permissions` para montar o menu.
	writeJSON(w, http.StatusOK, profilePayload(prof))
}

// handleListMembers lista membros no escopo da sessão (RLS).
// Opcionalmente filtra por nome quando o query param `q` é informado.
func (a *App) handleListMembers(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query().Get("q")
	b := boundsFromClaims(claims)
	var out []members.Member
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Members.List(r.Context(), tx, q)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": out})
}

// handleCreateMember cria um membro no escopo da sessão.
func (a *App) handleCreateMember(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in members.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var created *members.Member
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		created, err = a.Members.Create(r.Context(), tx, claims.TenantID, branchID, in, claims.UserID)
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"full_name": created.FullName})
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'member.created', 'members', $3, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, created.ID, payload)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleGetMember retorna um membro por ID no escopo da sessão.
func (a *App) handleGetMember(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	var m *members.Member
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		m, err = a.Members.Get(r.Context(), tx, id)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "member not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}
