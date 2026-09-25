package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/auth"
	"chosenerp/internal/finance"
	"chosenerp/internal/store"
)

// hasFinancePerm verifica se o papel do usuario tem a permissao financeira
// informada. O JWT carrega apenas o papel, entao a permissao e resolvida no
// banco (roles -> role_permissions -> permissions).
func (a *App) hasFinancePerm(ctx context.Context, tx pgx.Tx, claims *auth.Claims, perm string) (bool, error) {
	var ok bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM roles r
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE r.tenant_id = $1::uuid AND r.key = $2 AND p.key = $3)`,
		claims.TenantID, claims.Role, perm).Scan(&ok)
	return ok, err
}

func (a *App) handleListReconciliations(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []finance.Reconciliation
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListReconciliations(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reconciliations": out})
}

func (a *App) handleGetReconciliation(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var rec *finance.Reconciliation
	var items []finance.ReconciliationItem
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		rec, err = a.Finance.GetReconciliation(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return err
		}
		items, err = a.Finance.ListReconciliationItems(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "conciliacao nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reconciliation": rec, "items": items})
}

func (a *App) handleCreateReconciliation(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.CreateReconciliationInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	allowed := false
	var rec *finance.Reconciliation
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		allowed, err = a.hasFinancePerm(r.Context(), tx, claims, "finance.reconcile")
		if err != nil || !allowed {
			return err
		}
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		rec, err = a.Finance.CreateReconciliation(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.reconciliation.created', 'financial_reconciliations', $3, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, rec.ID, []byte(`{}`))
		return err
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeErr(w, http.StatusConflict, "ja existe conciliacao para um periodo sobreposto nesta filial")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !allowed {
		writeErr(w, http.StatusForbidden, "sem permissao para conciliar (finance.reconcile)")
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (a *App) handleConciliateReconciliation(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	allowed := false
	var rec *finance.Reconciliation
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		allowed, err = a.hasFinancePerm(r.Context(), tx, claims, "finance.reconcile")
		if err != nil || !allowed {
			return err
		}
		rec, err = a.Finance.Conciliate(r.Context(), tx, r.PathValue("id"), claims.UserID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.reconciliation.conciliated', 'financial_reconciliations', $3, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, rec.ID, []byte(`{}`))
		return err
	})
	if err != nil {
		if errors.Is(err, finance.ErrReconciliationClosed) {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "conciliacao nao encontrada")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !allowed {
		writeErr(w, http.StatusForbidden, "sem permissao para conciliar (finance.reconcile)")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (a *App) handleDeleteReconciliation(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !isAdmin(claims.Role) {
		writeErr(w, http.StatusForbidden, "apenas a Sede pode excluir uma conciliacao")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Finance.DeleteReconciliation(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "conciliacao nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
