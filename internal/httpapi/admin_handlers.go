package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chosenerp/internal/auth"
	"chosenerp/internal/org"
)

// superAdminClaims exige o papel super_admin (autorizacao de plataforma).
func (a *App) superAdminClaims(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return nil, false
	}
	if claims.Role != "super_admin" {
		writeErr(w, http.StatusForbidden, "apenas super_admin")
		return nil, false
	}
	return claims, true
}

// handleResetData limpa todo o dado operacional, mantendo a base (tenant,
// filiais, papeis, permissoes) e apenas o usuario super_admin. Pensado para
// reiniciar testes operacionais. Exige confirmacao explicita no corpo.
func (a *App) handleResetData(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.superAdminClaims(w, r); !ok {
		return
	}
	var in struct {
		Confirm string `json:"confirm"`
	}
	if err := readJSON(r, &in); err != nil || strings.ToUpper(strings.TrimSpace(in.Confirm)) != "RESET" {
		writeErr(w, http.StatusBadRequest, `envie {"confirm":"RESET"} para confirmar`)
		return
	}
	if _, err := a.Store.Pool().Exec(r.Context(), `SELECT reset_operational_data()`); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleListTenants lista as igrejas (super_admin) para a tela de onboarding.
func (a *App) handleListTenants(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.superAdminClaims(w, r); !ok {
		return
	}
	var out []org.AdminTenant
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		out, e = a.Org.ListTenants(r.Context(), tx)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tenants": out})
}

// handleCreateTenant cria uma igreja completa (tenant + Matriz + papeis +
// primeiro super_admin). O subdominio passa a funcionar automaticamente.
func (a *App) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.superAdminClaims(w, r); !ok {
		return
	}
	var in struct {
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		Plan          string `json:"plan"`
		AdminName     string `json:"admin_name"`
		AdminEmail    string `json:"admin_email"`
		AdminPassword string `json:"admin_password"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	in.AdminName = strings.TrimSpace(in.AdminName)
	in.AdminEmail = strings.TrimSpace(in.AdminEmail)
	if in.Name == "" || in.Slug == "" || in.AdminName == "" || in.AdminEmail == "" {
		writeErr(w, http.StatusBadRequest, "name, slug, admin_name e admin_email sao obrigatorios")
		return
	}
	if len(in.AdminPassword) < 8 {
		writeErr(w, http.StatusBadRequest, "a senha do admin deve ter ao menos 8 caracteres")
		return
	}

	hash, err := auth.HashPassword(in.AdminPassword)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	var tenantID string
	err = a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		id, e := a.Org.CreateTenant(r.Context(), tx, in.Name, in.Slug, in.Plan)
		if e != nil {
			return e
		}
		tenantID = id
		_, e = a.Org.CreateTenantAdmin(r.Context(), tx, tenantID, in.AdminEmail, hash, in.AdminName)
		return e
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				writeErr(w, http.StatusConflict, "slug ja esta em uso")
				return
			case "22023":
				writeErr(w, http.StatusBadRequest, pgErr.Message)
				return
			}
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	subdomain := in.Slug + "." + a.Config.PublicBaseDomain
	writeJSON(w, http.StatusCreated, map[string]any{
		"tenant_id":   tenantID,
		"slug":        in.Slug,
		"subdomain":   subdomain,
		"admin_email": in.AdminEmail,
	})
}
