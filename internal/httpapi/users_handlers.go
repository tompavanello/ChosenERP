package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chosenerp/internal/auth"
	"chosenerp/internal/store"
	"chosenerp/internal/users"
)

// isAdmin restringe a gestão de usuários à Sede do tenant. O RLS já isola o
// tenant; aqui é autorização por papel (o banco não distingue quem pode gerir).
func isAdmin(role string) bool { return role == "super_admin" || role == "admin_sede" }

func (a *App) adminClaims(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return nil, false
	}
	if !isAdmin(claims.Role) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return nil, false
	}
	return claims, true
}

func (a *App) handleListUsers(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	b := boundsFromClaims(claims)
	var out []users.User
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Users.List(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

func (a *App) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	var in users.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.Email = strings.TrimSpace(in.Email)
	in.FullName = strings.TrimSpace(in.FullName)
	if in.Email == "" || in.FullName == "" || in.RoleKey == "" {
		writeErr(w, http.StatusBadRequest, "email, full_name e role são obrigatórios")
		return
	}
	if len(in.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "a senha deve ter ao menos 8 caracteres")
		return
	}
	b := boundsFromClaims(claims)
	var created *users.User
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		created, err = a.Users.Create(r.Context(), tx, claims.TenantID, in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusBadRequest, "perfil (role) inválido")
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "P0002": // perfil inexistente no tenant (user_attach_to_tenant)
				writeErr(w, http.StatusBadRequest, "perfil (role) inválido")
				return
			case "23505": // e-mail já existe OU vínculo duplicado na igreja
				writeErr(w, http.StatusConflict, "e-mail já cadastrado nesta igreja")
				return
			}
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *App) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	var in users.UpdateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var updated *users.User
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		updated, err = a.Users.Update(r.Context(), tx, id, in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *App) handleResetUserPassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	var in struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil || len(in.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "a senha deve ter ao menos 8 caracteres")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Users.ResetPassword(r.Context(), tx, id, in.Password)
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleListRoles(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	b := boundsFromClaims(claims)
	var out []users.Role
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Users.ListRoles(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": out})
}

func (a *App) handleListPermissions(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	b := boundsFromClaims(claims)
	var out []users.Permission
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Users.ListPermissions(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"permissions": out})
}

// ---- MFA (TOTP) do próprio usuário ----

func (a *App) handleMFAStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	_, enabled, err := a.Auth.MFASecret(r.Context(), b, claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": enabled})
}

func (a *App) handleMFASetup(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	secret, err := a.Auth.SetupMFA(r.Context(), b, claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// O e-mail para rotular o autenticador vem do perfil.
	prof, err := a.Auth.Me(r.Context(), claims.UserID, claims.TenantID, claims.BranchID, claims.Role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "profile unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"secret":      secret,
		"otpauth_url": auth.TOTPURL(secret, prof.Email, "Chosen ERP"),
		"account":     prof.Email,
	})
}

func (a *App) handleMFAEnable(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Code string `json:"code"`
	}
	if err := readJSON(r, &in); err != nil || in.Code == "" {
		writeErr(w, http.StatusBadRequest, "code required")
		return
	}
	b := boundsFromClaims(claims)
	if err := a.Auth.EnableMFA(r.Context(), b, claims.UserID, in.Code); err != nil {
		if errors.Is(err, auth.ErrMFAInvalid) {
			writeErr(w, http.StatusBadRequest, "código inválido")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "enabled": true})
}

func (a *App) handleMFADisable(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	if err := a.Auth.DisableMFA(r.Context(), b, claims.UserID); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "enabled": false})
}
