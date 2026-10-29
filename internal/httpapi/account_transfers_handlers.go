package httpapi

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/finance"
	"chosenerp/internal/store"
)

func (a *App) handleListAccountTransfers(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	var out []finance.AccountTransfer
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListAccountTransfers(r.Context(), tx, q.Get("from"), q.Get("to"), q.Get("account_id"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transfers": out})
}

func (a *App) handleCreateAccountTransfer(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.CreateAccountTransferInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	var out *finance.AccountTransfer
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		out, err = a.Finance.CreateAccountTransfer(r.Context(), tx, claims.TenantID, claims.UserID, branchID, in)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.account_transfer.created', 'account_transfers', $3::uuid, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, out.ID, []byte(`{}`))
		return err
	})
	if err != nil {
		if errors.Is(err, finance.ErrAccountTransferNotFound) || store.IsNotFound(err) || errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "conta nao encontrada no escopo")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *App) handleDeleteAccountTransfer(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if err := a.Finance.DeleteAccountTransfer(r.Context(), tx, claims.TenantID, id); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.account_transfer.deleted', 'account_transfers', $3::uuid, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, id, []byte(`{}`))
		return err
	})
	if err != nil {
		if errors.Is(err, finance.ErrAccountTransferNotFound) || store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "transferencia nao encontrada")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleListAccountBalances(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	asOf := r.URL.Query().Get("as_of")
	var out *finance.AccountBalances
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListAccountBalances(r.Context(), tx, asOf)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}
