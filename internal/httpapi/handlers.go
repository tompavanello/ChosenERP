package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/auth"
	"chosenerp/internal/members"
	"chosenerp/internal/store"
)

// Config é a fatia da config global usada pelos handlers HTTP.
type Config struct {
	JWTSecret string
}

// boundsFromClaims converte claims em Bounds para RLS.
func boundsFromClaims(c *auth.Claims) store.Bounds {
	return store.Bounds{TenantID: c.TenantID, BranchID: c.BranchID, Role: c.Role}
}

// loginHandler autentica e retorna tokens.
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	prof, tokens, err := a.Auth.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tokens": tokens,
		"user": map[string]any{
			"id": prof.UserID, "email": prof.Email, "full_name": prof.FullName,
			"tenant_id": prof.TenantID, "branch_id": prof.BranchID, "role": prof.Role,
			"permissions": prof.Permissions,
		},
	})
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
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": claims.UserID, "tenant_id": claims.TenantID,
		"branch_id": claims.BranchID, "role": claims.Role,
	})
}

// handleListMembers lista membros no escopo da sessão (RLS).
func (a *App) handleListMembers(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []members.Member
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Members.List(r.Context(), tx)
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
		var err error
		created, err = a.Members.Create(r.Context(), tx, claims.TenantID, claims.BranchID, in)
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
