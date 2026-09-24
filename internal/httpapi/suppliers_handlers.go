package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/store"
	"chosenerp/internal/suppliers"
)

// ---- Fornecedores ----

func (a *App) handleListSuppliers(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []suppliers.Supplier
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Suppliers.List(r.Context(), tx, r.URL.Query().Get("q"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"suppliers": out})
}

func (a *App) handleCreateSupplier(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in suppliers.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	if in.Name == "" {
		writeErr(w, http.StatusBadRequest, "nome e obrigatorio")
		return
	}
	b := boundsFromClaims(claims)
	var s *suppliers.Supplier
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		s, err = a.Suppliers.Create(r.Context(), tx, claims.TenantID, claims.BranchID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (a *App) handleUpdateSupplier(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in suppliers.UpdateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	var s *suppliers.Supplier
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		s, err = a.Suppliers.Update(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "fornecedor nao encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (a *App) handleDeleteSupplier(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Suppliers.Delete(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "fornecedor nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
