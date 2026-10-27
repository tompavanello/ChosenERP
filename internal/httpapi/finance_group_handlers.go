package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/finance"
	"chosenerp/internal/store"
)

// handleListFinanceGroups lista os grupos de lancamentos do escopo.
func (a *App) handleListFinanceGroups(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	var out []finance.TransactionGroup
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListGroups(r.Context(), tx, q.Get("from"), q.Get("to"), q.Get("status"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": out})
}

func (a *App) handleGetFinanceGroup(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var g *finance.TransactionGroup
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var err error
		g, err = a.Finance.GetGroup(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if errors.Is(err, finance.ErrGroupNotFound) || store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "grupo nao encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (a *App) handleCreateFinanceGroup(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.CreateGroupInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	var g *finance.TransactionGroup
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var err error
		g, err = a.Finance.CreateGroup(r.Context(), tx, claims.TenantID, claims.UserID, in)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.group.created', 'financial_transaction_groups', $3::uuid, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, g.ID, []byte(`{}`))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "lancamento nao encontrado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (a *App) handleUpdateFinanceGroup(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.UpdateGroupInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	id := r.PathValue("id")
	var g *finance.TransactionGroup
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var err error
		g, err = a.Finance.UpdateGroup(r.Context(), tx, id, in)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.group.updated', 'financial_transaction_groups', $3::uuid, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, id, []byte(`{}`))
		return err
	})
	if err != nil {
		if errors.Is(err, finance.ErrGroupNotFound) || store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "grupo nao encontrado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (a *App) handleDeleteFinanceGroup(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if err := a.Finance.DeleteGroup(r.Context(), tx, id); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.group.deleted', 'financial_transaction_groups', $3::uuid, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, id, []byte(`{}`))
		return err
	})
	if err != nil {
		if errors.Is(err, finance.ErrGroupNotFound) || store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "grupo nao encontrado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleAddFinanceGroupMembers(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		TransactionIDs []string `json:"transaction_ids"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	id := r.PathValue("id")
	var g *finance.TransactionGroup
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var err error
		g, err = a.Finance.AddGroupMembers(r.Context(), tx, id, in.TransactionIDs)
		return err
	})
	if err != nil {
		if errors.Is(err, finance.ErrGroupNotFound) || store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "grupo nao encontrado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (a *App) handleRemoveFinanceGroupMember(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	txID := r.PathValue("txId")
	var g *finance.TransactionGroup
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var err error
		g, err = a.Finance.RemoveGroupMember(r.Context(), tx, id, txID)
		return err
	})
	if err != nil {
		if errors.Is(err, finance.ErrGroupNotFound) || store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "grupo nao encontrado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (a *App) handleReconcileFinanceGroup(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Reconciled bool `json:"reconciled"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	id := r.PathValue("id")
	var g *finance.TransactionGroup
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var err error
		g, err = a.Finance.ReconcileGroup(r.Context(), tx, id, in.Reconciled, claims.UserID)
		if err != nil {
			return err
		}
		action := "finance.group.reconciled"
		if !in.Reconciled {
			action = "finance.group.unreconciled"
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, $3, 'financial_transaction_groups', $4::uuid, $5
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, action, id, []byte(`{}`))
		return err
	})
	if err != nil {
		if errors.Is(err, finance.ErrGroupNotFound) || store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "grupo nao encontrado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

// handleGroupFromBankEntry cria um grupo a partir dos lancamentos selecionados e
// vincula a entrada do extrato bancario a ele.
func (a *App) handleGroupFromBankEntry(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.GroupFromBankEntryInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	importID := strings.TrimSpace(r.PathValue("id"))
	entryID := strings.TrimSpace(r.PathValue("entryId"))
	var g *finance.TransactionGroup
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var err error
		g, err = a.Finance.GroupFromBankEntry(r.Context(), tx, claims.TenantID, claims.UserID, importID, entryID, in)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.group.bank_linked', 'financial_transaction_groups', $3::uuid, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, g.ID, []byte(`{}`))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) || errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "entrada do extrato nao encontrada")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}
