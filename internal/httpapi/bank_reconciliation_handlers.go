package httpapi

import (
	"encoding/base64"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/finance"
	"chosenerp/internal/store"
)

func (a *App) handleListBankImports(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []finance.BankImport
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListBankImports(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"imports": out})
}

func (a *App) handleGetBankImport(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	id := r.PathValue("id")
	var imp *finance.BankImport
	var entries []finance.BankEntry
	var diverg []finance.BankDivergence
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		imp, err = a.Finance.GetBankImport(r.Context(), tx, id)
		if err != nil {
			return err
		}
		entries, err = a.Finance.ListBankEntries(r.Context(), tx, id)
		if err != nil {
			return err
		}
		diverg, err = a.Finance.ListBankDivergences(r.Context(), tx, id)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "importacao nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"import": imp, "entries": entries, "divergences": diverg})
}

func (a *App) handleImportBankStatement(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		AccountID string `json:"account_id"`
		Filename  string `json:"filename"`
		Data      string `json:"data"` // base64
	}
	if err := readJSONMax(r, &in, 12<<20); err != nil {
		writeErr(w, http.StatusBadRequest, "arquivo invalido ou muito grande (max 12MB)")
		return
	}
	raw, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "conteudo base64 invalido")
		return
	}
	b := boundsFromClaims(claims)
	allowed := false
	var imp *finance.BankImport
	err = a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		allowed, err = a.hasFinancePerm(r.Context(), tx, claims, "finance.reconcile")
		if err != nil || !allowed {
			return err
		}
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		imp, err = a.Finance.ImportBankStatement(r.Context(), tx, claims.TenantID, branchID, claims.UserID,
			finance.ImportBankInput{AccountID: in.AccountID, Filename: in.Filename, Raw: raw})
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !allowed {
		writeErr(w, http.StatusForbidden, "sem permissao para conciliar (finance.reconcile)")
		return
	}
	writeJSON(w, http.StatusCreated, imp)
}

func (a *App) handleIgnoreBankEntry(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	allowed := false
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		allowed, err = a.hasFinancePerm(r.Context(), tx, claims, "finance.reconcile")
		if err != nil || !allowed {
			return err
		}
		return a.Finance.IgnoreBankEntry(r.Context(), tx, r.PathValue("entryId"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "entrada nao encontrada ou ja tratada")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !allowed {
		writeErr(w, http.StatusForbidden, "sem permissao para conciliar (finance.reconcile)")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleGenerateBankEntry(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.GenerateBankEntryInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	allowed := false
	var created *finance.Transaction
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		allowed, err = a.hasFinancePerm(r.Context(), tx, claims, "finance.write")
		if err != nil || !allowed {
			return err
		}
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		created, err = a.Finance.GenerateFromBankEntry(r.Context(), tx, claims.TenantID, branchID, r.PathValue("entryId"), in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "entrada nao encontrada ou ja tratada")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !allowed {
		writeErr(w, http.StatusForbidden, "sem permissao para lancar (finance.write)")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"transaction": created})
}

func (a *App) handleDeleteBankImport(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !isAdmin(claims.Role) {
		writeErr(w, http.StatusForbidden, "apenas a Sede pode excluir uma importacao")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Finance.DeleteBankImport(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "importacao nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
