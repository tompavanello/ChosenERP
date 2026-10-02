package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chosenerp/internal/store"
	"chosenerp/internal/users"
)

// Gestao de acessos de UMA igreja pelo console da plataforma (suporte):
// listar, criar, editar (perfil/administrador), resetar senha. Sempre em
// WithSystem e com registro em audit_log.

// auditPlatformTx registra uma acao de plataforma sobre a igreja alvo.
func (a *App) auditPlatformTx(ctx context.Context, tx pgx.Tx, tenantID, actorID, action, entityID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
		VALUES ($1::uuid, NULLIF($2,'')::uuid, $3, 'users', NULLIF($4,'')::uuid, '{}'::jsonb)`,
		tenantID, actorID, action, entityID)
	return err
}

func (a *App) handleAdminListTenantUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	tenantID := r.PathValue("id")
	var out []users.User
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		out, e = a.Users.ListByTenant(r.Context(), tx, tenantID)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if out == nil {
		out = []users.User{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

func (a *App) handleAdminCreateTenantUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.platformAdminClaims(w, r)
	if !ok {
		return
	}
	tenantID := r.PathValue("id")
	var in users.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.Email = strings.TrimSpace(in.Email)
	in.FullName = strings.TrimSpace(in.FullName)
	if in.Email == "" || in.FullName == "" || in.RoleKey == "" {
		writeErr(w, http.StatusBadRequest, "email, full_name e role sao obrigatorios")
		return
	}
	if len(in.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "a senha deve ter ao menos 8 caracteres")
		return
	}
	var created *users.User
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		c, e := a.Users.CreateInTenant(r.Context(), tx, tenantID, in)
		if e != nil {
			return e
		}
		created = c
		return a.auditPlatformTx(r.Context(), tx, tenantID, claims.UserID, "platform.user.created", c.ID)
	})
	if err != nil {
		if writePlatformUserErr(w, err) {
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *App) handleAdminUpdateTenantUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.platformAdminClaims(w, r)
	if !ok {
		return
	}
	tenantID := r.PathValue("id")
	userID := r.PathValue("userId")
	var in users.UpdateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var updated *users.User
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		u, e := a.Users.UpdateInTenant(r.Context(), tx, tenantID, userID, in)
		if e != nil {
			return e
		}
		updated = u
		return a.auditPlatformTx(r.Context(), tx, tenantID, claims.UserID, "platform.user.updated", u.ID)
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "usuario nao encontrado nesta igreja")
			return
		}
		if writePlatformUserErr(w, err) {
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *App) handleAdminResetTenantUserPassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.platformAdminClaims(w, r)
	if !ok {
		return
	}
	tenantID := r.PathValue("id")
	userID := r.PathValue("userId")
	var in struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil || len(in.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "a senha deve ter ao menos 8 caracteres")
		return
	}
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		if e := a.Users.ResetPasswordInTenant(r.Context(), tx, tenantID, userID, in.Password); e != nil {
			return e
		}
		return a.auditPlatformTx(r.Context(), tx, tenantID, claims.UserID, "platform.user.password_reset", userID)
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "usuario nao encontrado nesta igreja")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleAdminListTenantRoles(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	tenantID := r.PathValue("id")
	var out []users.Role
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		out, e = a.Users.ListRolesByTenant(r.Context(), tx, tenantID)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if out == nil {
		out = []users.Role{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": out})
}

func (a *App) handleAdminListTenantBranches(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	tenantID := r.PathValue("id")
	var out []orgBranch
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		bs, e := a.Org.ListBranchesByTenant(r.Context(), tx, tenantID)
		for _, b := range bs {
			out = append(out, orgBranch{ID: b.ID, Name: b.Name, Kind: b.Kind, IsActive: b.IsActive})
		}
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if out == nil {
		out = []orgBranch{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"branches": out})
}

// orgBranch e a projecao minima de filial para o seletor do console.
type orgBranch struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	IsActive bool   `json:"is_active"`
}

func writePlatformUserErr(w http.ResponseWriter, err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	switch pgErr.Code {
	case "23505":
		writeErr(w, http.StatusConflict, "e-mail ja cadastrado nesta igreja (ou vinculo duplicado)")
	case "P0002":
		writeErr(w, http.StatusBadRequest, "perfil (role) invalido")
	case "22023":
		writeErr(w, http.StatusBadRequest, pgErr.Message)
	default:
		return false
	}
	return true
}
