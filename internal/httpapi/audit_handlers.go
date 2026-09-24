package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/audit"
	"chosenerp/internal/store"
)

func statusLabel(t string) string {
	if t == "income" {
		return "Entrada"
	}
	return "Saída"
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (a *App) handleListAudits(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []audit.Audit
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Audits.List(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"audits": out})
}

func (a *App) handleCreateAudit(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in audit.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo inválido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	var au *audit.Audit
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		au, err = a.Audits.Create(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, au)
}

func (a *App) handleGetAudit(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var au *audit.Audit
	var items []audit.Item
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		au, err = a.Audits.Get(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return err
		}
		items, err = a.Audits.ListItems(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "auditoria não encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"audit": au, "items": items})
}

func (a *App) handleMarkAudit(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in audit.MarkInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo inválido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Audits.Mark(r.Context(), tx, r.PathValue("id"), in, claims.UserID)
	})
	if err != nil {
		if errors.Is(err, audit.ErrClosed) {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "auditoria não encontrada")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleCloseAudit(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in audit.CloseInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo inválido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	var au *audit.Audit
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		au, err = a.Audits.Close(r.Context(), tx, r.PathValue("id"), in, claims.UserID)
		return err
	})
	if err != nil {
		if errors.Is(err, audit.ErrClosed) {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "auditoria não encontrada")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, au)
}

// handleExportAudit gera o PDF/CSV/XLSX do documento auditado (itens + assinatura).
func (a *App) handleExportAudit(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	format := first(r.URL.Query()["format"])
	b := boundsFromClaims(claims)
	var au *audit.Audit
	var items []audit.Item
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		au, err = a.Audits.Get(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return err
		}
		items, err = a.Audits.ListItems(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "auditoria não encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	rows := make([][]string, 0, len(items))
	for _, it := range items {
		desc := strOrEmpty(it.Description)
		cat := strOrEmpty(it.CategoryName)
		acc := strOrEmpty(it.AccountName)
		sup := strOrEmpty(it.SupplierName)
		pay := strOrEmpty(it.PaymentMethod)
		audited := "Não"
		if it.Audited {
			audited = "Sim"
		}
		rows = append(rows, []string{
			it.OccurredAt.Format("02/01/2006"), statusLabel(it.Type), cat, acc, sup, pay, desc,
			money(it.Amount), audited,
		})
	}
	auditedMark := "Em andamento"
	if au.Status == "fechada" {
		auditedMark = "FECHADA"
	}
	signer, role, closedAt, hash := "", "", "", ""
	if au.SignerName != nil {
		signer = *au.SignerName
	}
	if au.SignerRole != nil {
		role = *au.SignerRole
	}
	if au.ClosedAt != nil {
		closedAt = au.ClosedAt.Format("02/01/2006 15:04")
	}
	if au.SignatureHash != nil {
		hash = *au.SignatureHash
	}
	subtitle := "Período: " + au.PeriodStart + " a " + au.PeriodEnd +
		"  ·  Situação: " + auditedMark +
		"  ·  Auditados: " + strconv.Itoa(au.AuditedItems) + "/" + strconv.Itoa(au.TotalItems)

	writeReport(w, "auditoria-"+au.PeriodStart+"-"+au.PeriodEnd,
		"Auditoria financeira — "+au.Title, subtitle, format,
		[]exportSection{
			{Title: "Lançamentos do período", Headers: []string{"Data", "Tipo", "Conta contábil", "Conta bancária", "Fornecedor", "Forma pagto", "Descrição", "Valor", "Auditado"}, Rows: rows},
			{Title: "Totais", Headers: []string{"Indicador", "", "", "", "Valor"}, Rows: [][]string{
				{"Total do período", "", "", "", money(au.TotalAmount)},
				{"Total auditado", "", "", "", money(au.AuditedAmount)},
			}},
			{Title: "Fechamento / assinatura", Headers: []string{"Responsável", "Função", "Data/hora", "Hash", "", ""}, Rows: [][]string{
				{signer, role, closedAt, hash, "", ""},
			}},
		})
}

// handleDeleteAudit exclui a auditoria (e seus itens). Serve para descartar uma
// auditoria criada indevidamente; funciona mesmo se estiver fechada.
func (a *App) handleDeleteAudit(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Audits.Delete(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "auditoria não encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
