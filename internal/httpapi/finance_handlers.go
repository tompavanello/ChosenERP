package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/documents"
	"chosenerp/internal/finance"
	"chosenerp/internal/store"
)

func (a *App) handleListCategories(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []finance.Category
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListCategories(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": out})
}

func (a *App) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.CreateCategoryInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var c *finance.Category
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		c, err = a.Finance.CreateCategory(r.Context(), tx, claims.TenantID, claims.BranchID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (a *App) handleCreateTxn(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.CreateTxnInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Amount <= 0 {
		writeErr(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	b := boundsFromClaims(claims)
	var t *finance.Transaction
	var ref, token string
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		t, ref, token, err = a.Finance.Create(r.Context(), tx, claims.TenantID, claims.BranchID, in)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.tx.inserted', 'financial_transactions', $3, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, t.ID, []byte(`{}`))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"transaction": t, "receipt_ref": ref, "receipt_token": token,
	})
}

func (a *App) handleListTxn(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	kind := r.URL.Query().Get("type")
	b := boundsFromClaims(claims)
	var out []finance.Transaction
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.List(r.Context(), tx, kind)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transactions": out})
}

func (a *App) handleBalance(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	kind := r.URL.Query().Get("type")
	b := boundsFromClaims(claims)
	var bal finance.Balance
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		bal, err = a.Finance.SumBalance(r.Context(), tx, kind)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, bal)
}

func (a *App) handleGetDocumentByToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	token := r.PathValue("token")
	b := boundsFromClaims(claims)
	var doc *documents.Document
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		doc, err = a.Documents.GetByToken(r.Context(), tx, token)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "document not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (a *App) handleIssueCard(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	memberID := r.PathValue("id")
	b := boundsFromClaims(claims)
	var ref string
	var name string
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		m, err := a.Members.Get(r.Context(), tx, memberID)
		if err != nil {
			return err
		}
		name = m.FullName
		ref, err = a.Finance.NewMembershipCard(r.Context(), tx, claims.TenantID, claims.BranchID, memberID, m.FullName)
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
	writeJSON(w, http.StatusCreated, map[string]any{"card_ref": ref, "member": name})
}
