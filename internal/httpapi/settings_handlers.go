package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chosenerp/internal/auth"
	"chosenerp/internal/org"
	"chosenerp/internal/store"
)

// ---- Painel consolidado Sede > Filiais ----

func (a *App) handleConsolidatedReport(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	b := boundsFromClaims(claims)
	var out []org.BranchSummary
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Org.Consolidated(r.Context(), tx, q.Get("from"), q.Get("to"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var members, visitors int
	var income, expense float64
	for _, s := range out {
		members += s.MemberCount
		visitors += s.VisitorCount
		income += s.Income
		expense += s.Expense
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"branches": out,
		"totals": map[string]any{
			"member_count": members, "visitor_count": visitors,
			"income": income, "expense": expense, "net": income - expense,
		},
	})
}

// ---- Perfil do próprio usuário ----

func (a *App) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.FullName = strings.TrimSpace(in.FullName)
	in.Email = strings.TrimSpace(in.Email)
	b := boundsFromClaims(claims)
	if err := a.Auth.UpdateProfile(r.Context(), b, claims.UserID, in.FullName, in.Email); err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeErr(w, http.StatusConflict, "e-mail já cadastrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	prof, err := a.Auth.Me(r.Context(), claims.UserID, claims.TenantID, claims.BranchID, claims.Role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "profile unavailable")
		return
	}
	writeJSON(w, http.StatusOK, profilePayload(prof))
}

func (a *App) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := readJSON(r, &in); err != nil || in.CurrentPassword == "" || len(in.NewPassword) < 8 {
		writeErr(w, http.StatusBadRequest, "a nova senha deve ter ao menos 8 caracteres")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Auth.ChangePassword(r.Context(), b, claims.UserID, in.CurrentPassword, in.NewPassword)
	if err != nil {
		if errors.Is(err, auth.ErrWrongPassword) {
			writeErr(w, http.StatusBadRequest, "senha atual incorreta")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Configuração do tenant ----

func (a *App) handleGetTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var t *org.Tenant
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		t, err = a.Org.GetTenant(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *App) handleUpdateTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	var in org.TenantInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var t *org.Tenant
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		t, err = a.Org.UpdateTenant(r.Context(), tx, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// ---- Branding público do tenant (sem auth) ----

// handlePublicTenant devolve o branding da igreja pelo slug, para a tela de
// login do subdomínio. Resposta genérica e sem listagem — evita enumeração.
func (a *App) handlePublicTenant(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	var t struct {
		Name       string
		Slug       string
		LogoURL    *string
		BrandColor *string
		FaviconURL *string
	}
	err := a.Store.Pool().QueryRow(r.Context(),
		`SELECT name, slug, logo_url, brand_color, favicon_url FROM public_tenant($1)`, slug).
		Scan(&t.Name, &t.Slug, &t.LogoURL, &t.BrandColor, &t.FaviconURL)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name": t.Name, "slug": t.Slug, "logo_url": t.LogoURL,
		"brand_color": t.BrandColor, "favicon_url": t.FaviconURL,
	})
}

// ---- Filiais ----

func (a *App) handleCreateBranch(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	var in org.BranchInput
	if err := readJSON(r, &in); err != nil || strings.TrimSpace(in.Name) == "" {
		writeErr(w, http.StatusBadRequest, "name é obrigatório")
		return
	}
	b := boundsFromClaims(claims)
	var out *org.Branch
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Org.CreateBranch(r.Context(), tx, claims.TenantID, in)
		return err
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeErr(w, http.StatusConflict, "já existe uma filial com este identificador (slug)")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *App) handleUpdateBranch(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	var in org.BranchInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var out *org.Branch
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Org.UpdateBranch(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "filial não encontrada")
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeErr(w, http.StatusConflict, "já existe uma filial com este identificador (slug)")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleDeleteBranch(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Org.DeleteBranch(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		switch {
		case errors.Is(err, org.ErrBranchInUse):
			writeErr(w, http.StatusConflict, err.Error())
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "filial não encontrada")
		default:
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
