package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chosenerp/internal/auth"
	"chosenerp/internal/delivery"
	"chosenerp/internal/documents"
	"chosenerp/internal/finance"
	"chosenerp/internal/org"
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

func (a *App) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.UpdateCategoryInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	var c *finance.Category
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		c, err = a.Finance.UpdateCategory(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "conta contabil nao encontrada")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *App) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Finance.DeleteCategory(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if errors.Is(err, finance.ErrCategoryInUse) {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "conta contabil nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Grupos de contas do plano de contas ----

func (a *App) handleListCategoryGroups(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []finance.CategoryGroup
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListCategoryGroups(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": out})
}

func (a *App) handleCreateCategoryGroup(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.CreateCategoryGroupInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeErr(w, http.StatusBadRequest, "name required")
		return
	}
	b := boundsFromClaims(claims)
	var g *finance.CategoryGroup
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		g, err = a.Finance.CreateCategoryGroup(r.Context(), tx, claims.TenantID, claims.BranchID, in)
		return err
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeErr(w, http.StatusConflict, "ja existe um grupo com este nome")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (a *App) handleUpdateCategoryGroup(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.UpdateCategoryGroupInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var g *finance.CategoryGroup
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		g, err = a.Finance.UpdateCategoryGroup(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "grupo nao encontrado")
			return
		}
		if isUniqueViolation(err) {
			writeErr(w, http.StatusConflict, "ja existe um grupo com este nome")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (a *App) handleDeleteCategoryGroup(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Finance.DeleteCategoryGroup(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "grupo nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// isUniqueViolation reconhece o erro 23505 do Postgres (nome duplicado).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// errBranchOutOfScope indica que o usuario tentou gravar em uma filial fora do
// seu contexto (so a Sede no escopo "Todas" pode escolher a filial).
var errBranchOutOfScope = errors.New("filial fora do escopo do usuario")

// writeFinanceErr traduz as travas de conciliacao/periodo/auditoria em 409.
func writeFinanceErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, finance.ErrTransactionReconciled):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, finance.ErrScopeReconciled):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, finance.ErrTransactionVoided):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, finance.ErrPeriodClosed):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, finance.ErrAuditClosed):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, finance.ErrTransactionGrouped):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, finance.ErrGroupedReconcile):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, finance.ErrGroupReconciled):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, finance.ErrGroupDateMismatch):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, finance.ErrAccountTransferReconciled),
		errors.Is(err, finance.ErrAccountTransferGrouped),
		errors.Is(err, finance.ErrTransactionIsTransfer):
		writeErr(w, http.StatusConflict, err.Error())
	default:
		var pending finance.ErrPendingReconciliation
		if errors.As(err, &pending) {
			writeErr(w, http.StatusConflict, pending.Error())
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
	}
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
		writeErr(w, http.StatusBadRequest, finance.ErrAmountNotPositive.Error())
		return
	}
	if in.AccountID == nil || strings.TrimSpace(*in.AccountID) == "" {
		writeErr(w, http.StatusBadRequest, finance.ErrAccountRequired.Error())
		return
	}
	b := boundsFromClaims(claims)
	allowed := !in.Reconciled
	var t *finance.Transaction
	var ref, token string
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		if in.Reconciled {
			ok, err := a.hasFinancePerm(r.Context(), tx, claims, "finance.reconcile")
			if err != nil {
				return err
			}
			allowed = ok
			if !ok {
				return nil
			}
		}
		in.ActorID = claims.UserID
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		// A Sede no escopo "Todas" pode inserir em uma filial especifica (ex.:
		// entre linhas de outra filial). Usuarios de filial so gravam na propria.
		if in.BranchID != nil {
			requested := strings.TrimSpace(*in.BranchID)
			if requested != "" && requested != branchID {
				if claims.BranchID != "" || !auth.IsHQRole(claims.Role) {
					return errBranchOutOfScope
				}
				var exists bool
				if err := tx.QueryRow(r.Context(), `
					SELECT EXISTS (
						SELECT 1 FROM branches WHERE id = $1::uuid AND is_active
					)`, requested).Scan(&exists); err != nil {
					return err
				}
				if !exists {
					return errors.New("filial invalida ou inativa")
				}
				branchID = requested
			}
		}
		t, _, ref, token, err = a.Finance.Create(r.Context(), tx, claims.TenantID, branchID, in)
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
		if errors.Is(err, errBranchOutOfScope) {
			writeErr(w, http.StatusForbidden, err.Error())
			return
		}
		writeFinanceErr(w, err)
		return
	}
	if !allowed {
		writeErr(w, http.StatusForbidden, "sem permissao para conciliar (finance.reconcile)")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"transaction": t, "receipt_ref": ref, "receipt_token": token,
	})
}

// handleUpdateTxn edita um lancamento EM VIGOR (preserva id, entry_seq, recibo e
// anexos). A posicao (entry_seq) nao muda por aqui; reposicionar e via
// POST /finance/transactions/reorder.
func (a *App) handleUpdateTxn(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var in finance.UpdateTxnInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Amount <= 0 {
		writeErr(w, http.StatusBadRequest, finance.ErrAmountNotPositive.Error())
		return
	}
	if in.AccountID == nil || strings.TrimSpace(*in.AccountID) == "" {
		writeErr(w, http.StatusBadRequest, finance.ErrAccountRequired.Error())
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		if err := a.Finance.Update(r.Context(), tx, claims.TenantID, id, in); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.tx.updated', 'financial_transactions', $3::uuid, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, id, []byte(`{}`))
		return err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "lancamento nao encontrado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleReconcileTxn concilia/desconcilia um unico lancamento.
func (a *App) handleReconcileTxn(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Reconciled bool `json:"reconciled"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	allowed := false
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		allowed, err = a.hasFinancePerm(r.Context(), tx, claims, "finance.reconcile")
		if err != nil || !allowed {
			return err
		}
		if err := a.Finance.ReconcileTransaction(r.Context(), tx, id, in.Reconciled, claims.UserID); err != nil {
			return err
		}
		action := "finance.tx.unreconciled"
		if in.Reconciled {
			action = "finance.tx.reconciled"
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, $3, 'financial_transactions', $4::uuid, $5
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, action, id, []byte(`{}`))
		return err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "lancamento nao encontrado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	if !allowed {
		writeErr(w, http.StatusForbidden, "sem permissao para conciliar (finance.reconcile)")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleReconcileTxnBatch concilia/desconcilia varios lancamentos de uma vez.
func (a *App) handleReconcileTxnBatch(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		TransactionIDs []string `json:"transaction_ids"`
		Reconciled     bool     `json:"reconciled"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido")
		return
	}
	if len(in.TransactionIDs) == 0 {
		writeErr(w, http.StatusBadRequest, "nenhum lancamento informado")
		return
	}
	if len(in.TransactionIDs) > 500 {
		writeErr(w, http.StatusBadRequest, "limite de 500 lancamentos por vez")
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
		if err := a.Finance.ReconcileTransactions(r.Context(), tx, in.TransactionIDs, in.Reconciled, claims.UserID); err != nil {
			return err
		}
		action := "finance.tx.unreconciled"
		if in.Reconciled {
			action = "finance.tx.reconciled"
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, payload)
			SELECT $1, $2::uuid, $3, 'financial_transactions', $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, action, []byte(`{}`))
		return err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "lancamento nao encontrado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	if !allowed {
		writeErr(w, http.StatusForbidden, "sem permissao para conciliar (finance.reconcile)")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleCreateTxnBatch cria varios lancamentos em uma unica chamada (grid de
// entrada rapida). Linhas invalidas sao devolvidas individualmente, sem
// derrubar as demais.
func (a *App) handleCreateTxnBatch(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Transactions []finance.CreateTxnInput `json:"transactions"`
	}
	if err := readJSONMax(r, &in, 8<<20); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido")
		return
	}
	if len(in.Transactions) == 0 {
		writeErr(w, http.StatusBadRequest, "nenhum lancamento enviado")
		return
	}
	if len(in.Transactions) > 500 {
		writeErr(w, http.StatusBadRequest, "limite de 500 lancamentos por lote")
		return
	}
	b := boundsFromClaims(claims)
	needReconcile := false
	for i := range in.Transactions {
		if in.Transactions[i].Reconciled {
			needReconcile = true
		}
		in.Transactions[i].ActorID = claims.UserID
	}
	var res finance.ImportResult
	allowed := !needReconcile
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		if needReconcile {
			ok, err := a.hasFinancePerm(r.Context(), tx, claims, "finance.reconcile")
			if err != nil {
				return err
			}
			allowed = ok
			if !ok {
				return nil
			}
		}
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		res, err = a.Finance.CreateBatch(r.Context(), tx, claims.TenantID, branchID, in.Transactions)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.tx.batch_inserted', 'financial_transactions', NULL, $3
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, []byte(`{}`))
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
	writeJSON(w, http.StatusOK, res)
}

// handleReorderTxns renumera uma lista de lancamentos da MESMA conta e data,
// na ordem enviada, para alinhar a sequencia com o extrato. Somente entry_seq e
// alterado (fora da hash-chain).
func (a *App) handleReorderTxns(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		OrderedIDs []string `json:"ordered_ids"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(in.OrderedIDs) == 0 {
		writeErr(w, http.StatusBadRequest, "nenhum lancamento informado")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Finance.Reorder(r.Context(), tx, in.OrderedIDs)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "lancamento nao encontrado no escopo")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleImportTransactions importa lancamentos em lote via CSV ou planilha
// (XLSX). Aceita o CSV com cabecalho padrao (`csv`) ou um arquivo em base64 com
// mapeamento de colunas e linha inicial (`data`/`filename`/`start_row`/`mapping`).
func (a *App) handleImportTransactions(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		CSV              string         `json:"csv"`
		Data             string         `json:"data"`
		Filename         string         `json:"filename"`
		StartRow         int            `json:"start_row"`
		Mapping          map[string]int `json:"mapping"`
		DefaultAccountID string         `json:"default_account_id"`
	}
	if err := readJSONMax(r, &in, 8<<20); err != nil {
		writeErr(w, http.StatusBadRequest, "arquivo invalido ou muito grande (max 8MB)")
		return
	}
	b := boundsFromClaims(claims)
	var res finance.ImportResult
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		if in.CSV != "" {
			var e error
			res, e = a.Finance.ImportCSV(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in.DefaultAccountID, in.CSV)
			return e
		}
		if in.Data == "" {
			return errors.New("envie o CSV ou um arquivo")
		}
		raw, e := base64.StdEncoding.DecodeString(in.Data)
		if e != nil {
			return errors.New("arquivo em base64 invalido")
		}
		rows, e := finance.ParseSheet(raw, in.Filename)
		if e != nil {
			return e
		}
		res, e = a.Finance.ImportRecords(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in.DefaultAccountID, rows, in.StartRow, in.Mapping)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handlePreviewTransactions le a planilha e devolve as primeiras linhas para o
// usuario mapear as colunas antes de importar.
func (a *App) handlePreviewTransactions(w http.ResponseWriter, r *http.Request) {
	if _, ok := claimsFrom(r.Context()); !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Data     string `json:"data"`
		Filename string `json:"filename"`
	}
	if err := readJSONMax(r, &in, 8<<20); err != nil || in.Data == "" {
		writeErr(w, http.StatusBadRequest, "envie o arquivo")
		return
	}
	raw, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "arquivo em base64 invalido")
		return
	}
	rows, err := finance.ParseSheet(raw, in.Filename)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(rows) > 50 {
		rows = rows[:50]
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

// handleVoidTxn anula (estorna) um lancamento. O registro permanece no
// historico; os relatorios deixam de conta-lo.
func (a *App) handleVoidTxn(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var in struct {
		Reason string `json:"reason"`
	}
	_ = readJSON(r, &in) // motivo e opcional
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		if err := a.Finance.Void(r.Context(), tx, id, in.Reason, claims.UserID); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.tx.voided', 'financial_transactions', $3::uuid, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, id, []byte(`{}`))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) || errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "lancamento nao encontrado ou ja estornado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleSettleTxn marca/desmarca a quitacao de um lancamento (contas a pagar).
func (a *App) handleSettleTxn(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var in struct {
		Paid bool `json:"paid"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Finance.Settle(r.Context(), tx, id, in.Paid)
	})
	if err != nil {
		if store.IsNotFound(err) || errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "lancamento nao encontrado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleGetSplit devolve o estado do split de repasses e as regras da igreja.
func (a *App) handleGetSplit(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var enabled bool
	var rules []finance.SplitRule
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		enabled, rules, e = a.Finance.GetSplit(r.Context(), tx)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rules == nil {
		rules = []finance.SplitRule{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": enabled, "rules": rules})
}

// handlePutSplit grava o liga/desliga + regras do split de repasses.
func (a *App) handlePutSplit(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Enabled bool                     `json:"enabled"`
		Rules   []finance.SplitRuleInput `json:"rules"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		return a.Finance.SetSplit(r.Context(), tx, claims.TenantID, in.Enabled, in.Rules)
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleDeleteTxn exclui DEFINITIVAMENTE um lancamento (sem deixar estorno) e
// recalcula a hash-chain do tenant. Bloqueado quando preso a auditoria fechada.
func (a *App) handleDeleteTxn(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		if err := a.Finance.Delete(r.Context(), tx, id); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `
			INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, payload)
			SELECT $1, $2::uuid, 'finance.tx.deleted', 'financial_transactions', $3::uuid, $4
			FROM users WHERE id = $2`, claims.TenantID, claims.UserID, id, []byte(`{}`))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) || errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "lancamento nao encontrado")
			return
		}
		writeFinanceErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleListTxnEvents devolve o rateio do lancamento por evento.
func (a *App) handleListTxnEvents(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []finance.EventAllocation
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListTransactionEvents(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"allocations": out})
}

// parseFinDate valida uma data opcional de filtro (AAAA-MM-DD).
func parseFinDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return "", errors.New("data invalida (use AAAA-MM-DD)")
	}
	return s, nil
}

func (a *App) handleListTxn(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	kind := q.Get("type")
	from, err := parseFinDate(q.Get("from"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	to, err := parseFinDate(q.Get("to"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if from != "" && to != "" && from > to {
		writeErr(w, http.StatusBadRequest, "periodo invalido: data inicial maior que a final")
		return
	}
	b := boundsFromClaims(claims)
	var out []finance.Transaction
	err = a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.List(r.Context(), tx, kind, from, to)
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

// handleIssueCard emite (ou recupera) a carteirinha do membro.
// Idempotente: chamadas repetidas devolvem a MESMA carteirinha. Responde 200 (e
// nao 201) porque passou a ser get-or-create - 201 num no-op seria mentira.
// As chaves "card_ref" e "member" continuam existindo (backlog #42); "token" e
// nova. O token nao entra em members.List de proposito: /public/card/{token} e
// nao autenticado, entao expor N tokens numa listagem seria expor N URLs
// permanentes de uma vez. Quem quer o token pede por membro.
func (a *App) handleIssueCard(w http.ResponseWriter, r *http.Request) {
	a.respondCard(w, r, true)
}

// handleGetCard devolve a carteirinha ja emitida, sem efeito colateral.
func (a *App) handleGetCard(w http.ResponseWriter, r *http.Request) {
	a.respondCard(w, r, false)
}

func (a *App) respondCard(w http.ResponseWriter, r *http.Request, issue bool) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	memberID := r.PathValue("id")
	b := boundsFromClaims(claims)
	var ref, token, name string
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		m, err := a.Members.Get(r.Context(), tx, memberID)
		if err != nil {
			return err
		}
		name = m.FullName
		if issue {
			ref, token, err = a.Finance.IssueMembershipCard(r.Context(), tx, claims.TenantID, claims.BranchID, memberID, m.FullName)
		} else {
			ref, token, err = a.Finance.GetMembershipCard(r.Context(), tx, memberID)
		}
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
	writeJSON(w, http.StatusOK, map[string]any{"card_ref": ref, "token": token, "member": name})
}

// ---- Branches (para repasses e selecao de filial) ----

type branchDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Kind string `json:"kind"`
}

func (a *App) handleListBranches(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	out := []org.Branch{}
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Org.ListBranches(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Enriquece o numero conectado das filiais com WhatsApp aberto que ainda
	// nao tem o numero salvo. Best-effort: se a Evolution estiver fora, o grid
	// apenas nao mostra o numero (nao quebra a listagem).
	if a.Config.EvolutionAPIURL != "" && a.Config.EvolutionAPIKey != "" {
		client := a.evolution()
		for i := range out {
			br := &out[i]
			// A instancia Evolution e nomeada com o id da filial.
			if br.WhatsAppStatus != "connected" || br.WhatsAppNumber != "" {
				continue
			}
			info, err := client.FetchInstance(r.Context(), br.ID)
			if err != nil || info.ConnectionStatus != "open" {
				continue
			}
			num := delivery.ConnectedNumber(info)
			if num == "" {
				continue
			}
			br.WhatsAppNumber = num
			_ = a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
				return a.Org.SetBranchWhatsApp(r.Context(), tx, br.ID, br.ID, "connected", num)
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"branches": out})
}

// ---- Repasses entre filiais ----

func (a *App) handleCreateTransfer(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.CreateTransferInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Amount <= 0 {
		writeErr(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	if in.FromBranchID == "" || in.ToBranchID == "" {
		writeErr(w, http.StatusBadRequest, "from_branch_id and to_branch_id required")
		return
	}
	b := boundsFromClaims(claims)
	var t *finance.Transfer
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		t, err = a.Finance.CreateTransfer(r.Context(), tx, claims.TenantID, claims.BranchID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (a *App) handleListTransfers(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []finance.Transfer
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListTransfers(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transfers": out})
}

// ---- Doacoes recorrentes ----

func (a *App) handleListRecurring(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []finance.Recurring
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListRecurring(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recurring": out})
}

func (a *App) handleCreateRecurring(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.CreateRecurringInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var rec *finance.Recurring
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		rec, err = a.Finance.CreateRecurring(r.Context(), tx, claims.TenantID, branchID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (a *App) handleUpdateRecurring(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.UpdateRecurringInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var rec *finance.Recurring
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		rec, err = a.Finance.UpdateRecurring(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

// ---- Contas bancarias ----

func (a *App) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []finance.Account
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListAccounts(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": out})
}

func (a *App) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.CreateAccountInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	b := boundsFromClaims(claims)
	var acct *finance.Account
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		acct, err = a.Finance.CreateAccount(r.Context(), tx, claims.TenantID, branchID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, acct)
}

func (a *App) handleUpdateAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in finance.UpdateAccountInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var acct *finance.Account
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		acct, err = a.Finance.UpdateAccount(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "account not found")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, acct)
}

func (a *App) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Finance.DeleteAccount(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if errors.Is(err, finance.ErrAccountInUse) {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "conta nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Anexos de lancamentos ----

func (a *App) saveFinanceUpload(w http.ResponseWriter, r *http.Request) (fileName, diskPath, contentType string, size int64, ok bool) {
	// Limite 50 MB em uploads multipart. MaxBytesReader corta a leitura no limite
	// antes de bufferizar (ParseMultipartForm so limita a parte em memoria).
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, "upload muito grande (max 50MB)")
		return "", "", "", 0, false
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "arquivo nao enviado")
		return "", "", "", 0, false
	}
	defer file.Close()

	// Cria diretorio de uploads se nao existir.
	if err := os.MkdirAll(a.Config.UploadDir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, "nao foi possivel criar diretorio de uploads")
		return "", "", "", 0, false
	}

	fileName = sanitizeFilename(header.Filename)
	randBytes := make([]byte, 12)
	_, _ = rand.Read(randBytes)
	diskName := hex.EncodeToString(randBytes) + "_" + fileName
	diskPath = filepath.Join(a.Config.UploadDir, diskName)

	dst, err := os.Create(diskPath)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "nao foi possivel salvar arquivo")
		return "", "", "", 0, false
	}
	n, err := io.Copy(dst, file)
	dst.Close()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "erro ao salvar arquivo")
		return "", "", "", 0, false
	}

	size = int64(n)
	contentType = header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(fileName))
	}
	return fileName, diskPath, contentType, size, true
}

func (a *App) handleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	transactionID := r.PathValue("id")
	b := boundsFromClaims(claims)

	fileName, diskPath, contentType, size, ok := a.saveFinanceUpload(w, r)
	if !ok {
		return
	}
	diskName := filepath.Base(diskPath)

	in := finance.CreateAttachmentInput{
		FileName:    fileName,
		FileURL:     "/api/v1/attachments/" + diskName,
		ContentType: &contentType,
		FileSize:    size,
	}

	var att *finance.Attachment
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		if e := a.enforceQuota(r.Context(), tx, claims.TenantID, "storage", size); e != nil {
			_ = os.Remove(diskPath) // nao deixa arquivo orfao quando o plano bloqueia
			return e
		}
		var err error
		// Verifica que a transacao pertence ao escopo antes de anexar.
		var txExists bool
		err = tx.QueryRow(r.Context(), `
			SELECT EXISTS(SELECT 1 FROM financial_transactions
			              WHERE id = $1::uuid AND tenant_id = $2 AND branch_id = $3)`,
			transactionID, claims.TenantID, claims.BranchID).Scan(&txExists)
		if err != nil {
			return err
		}
		if !txExists && claims.BranchID != "" {
			// Tambem permite anexar de escopo Sede (consulta sem branch_id).
			err = tx.QueryRow(r.Context(), `
				SELECT EXISTS(SELECT 1 FROM financial_transactions
			              WHERE id = $1::uuid AND tenant_id = $2)`,
				transactionID, claims.TenantID).Scan(&txExists)
		}
		if err != nil || !txExists {
			return pgx.ErrNoRows
		}
		att, err = a.Finance.CreateAttachment(r.Context(), tx, claims.TenantID, claims.BranchID, transactionID, in)
		return err
	})
	if err != nil {
		_ = os.Remove(diskPath)
		if writeQuotaErr(w, err) {
			return
		}
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "transaction not found")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, att)
}

func (a *App) handleListAttachments(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	transactionID := r.PathValue("id")
	b := boundsFromClaims(claims)
	var out []finance.Attachment
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Finance.ListAttachments(r.Context(), tx, transactionID)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"attachments": out})
}

// handleDownloadAttachment serve, sem autenticacao, SOMENTE as fotos de membro
// (nome opaco de 32 hex + extensao de imagem). E o que a carteirinha publica
// precisa exibir. Qualquer outro arquivo (ex.: comprovante financeiro) e
// servido apenas por rotas autenticadas - ver handleDownloadFinanceAttachment.
func (a *App) handleDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.PathValue("filename"))
	if !isOwnUploadName(filename) {
		// Nao e uma foto de membro: nao existe publicamente.
		writeErr(w, http.StatusNotFound, "arquivo nao encontrado")
		return
	}
	a.serveUploadedFile(w, r, filename)
}

// handleDownloadFinanceAttachment serve um anexo do financeiro, exigindo
// permissao e escopo: so entrega o arquivo se ele estiver registrado em
// financial_attachments do tenant (e filial) do solicitante (via RLS).
func (a *App) handleDownloadFinanceAttachment(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	filename := filepath.Base(r.PathValue("filename"))
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		writeErr(w, http.StatusBadRequest, "arquivo invalido")
		return
	}
	b := boundsFromClaims(claims)
	var exists bool
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return tx.QueryRow(r.Context(), `
			SELECT EXISTS(SELECT 1 FROM financial_attachments
			  WHERE file_url = $1 AND tenant_id = $2)`,
			"/api/v1/attachments/"+filename, claims.TenantID).Scan(&exists)
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "erro ao acessar arquivo")
		return
	}
	if !exists {
		writeErr(w, http.StatusNotFound, "arquivo nao encontrado")
		return
	}
	a.serveUploadedFile(w, r, filename)
}

// serveUploadedFile entrega um arquivo do diretorio de uploads (nome ja
// higienizado com filepath.Base). Compartilhado pelas rotas publica (fotos) e
// autenticada (financeiro).
func (a *App) serveUploadedFile(w http.ResponseWriter, r *http.Request, filename string) {
	diskPath := filepath.Join(a.Config.UploadDir, filename)
	if _, err := os.Stat(diskPath); err != nil {
		if os.IsNotExist(err) {
			writeErr(w, http.StatusNotFound, "arquivo nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, "erro ao acessar arquivo")
		return
	}
	ct := mime.TypeByExtension(filepath.Ext(filename))
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	// nosniff impede o browser de "adivinhar" um tipo executavel para um arquivo
	// enviado por usuario; o cache e seguro porque o nome e unico por upload.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	// Imagens podem ser exibidas inline (foto em <img>); o resto e forçado a
	// download, evitando execucao/renderizacao de HTML/SVG na mesma origem.
	if strings.HasPrefix(ct, "image/") {
		w.Header().Set("Content-Disposition", "inline; filename="+strconv.Quote(filename))
	} else {
		w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(filename))
	}
	http.ServeFile(w, r, diskPath)
}

// sanitizeFilename remove componentes de caminho e limita o tamanho.
func sanitizeFilename(name string) string {
	base := filepath.Base(name)
	if len(base) > 200 {
		base = base[:200]
	}
	return base
}
