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

	"github.com/jackc/pgx/v5"

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
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
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
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"transaction": t, "receipt_ref": ref, "receipt_token": token,
	})
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
		CSV      string         `json:"csv"`
		Data     string         `json:"data"`
		Filename string         `json:"filename"`
		StartRow int            `json:"start_row"`
		Mapping  map[string]int `json:"mapping"`
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
			res, e = a.Finance.ImportCSV(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in.CSV)
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
		res, e = a.Finance.ImportRecords(r.Context(), tx, claims.TenantID, branchID, claims.UserID, rows, in.StartRow, in.Mapping)
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
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "lancamento nao encontrado ou ja estornado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
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

func (a *App) handleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	transactionID := r.PathValue("id")
	b := boundsFromClaims(claims)

	// Limite 50 MB em uploads multipart.
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, "upload muito grande (max 50MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "arquivo nao enviado")
		return
	}
	defer file.Close()

	// Cria diretorio de uploads se nao existir.
	if err := os.MkdirAll(a.Config.UploadDir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, "nao foi possivel criar diretorio de uploads")
		return
	}

	filename := sanitizeFilename(header.Filename)
	randBytes := make([]byte, 12)
	_, _ = rand.Read(randBytes)
	diskName := hex.EncodeToString(randBytes) + "_" + filename
	diskPath := filepath.Join(a.Config.UploadDir, diskName)

	dst, err := os.Create(diskPath)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "nao foi possivel salvar arquivo")
		return
	}
	n, err := io.Copy(dst, file)
	dst.Close()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "erro ao salvar arquivo")
		return
	}

	size := int64(n)
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(filename))
	}

	in := finance.CreateAttachmentInput{
		FileName:    filename,
		FileURL:     "/api/v1/attachments/" + diskName,
		ContentType: &contentType,
		FileSize:    size,
	}

	var att *finance.Attachment
	err = a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
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

// handleDownloadAttachment serve o arquivo do disco. O nome do arquivo (uuid)
// e opaco - nao expoe caminho real nem credenciais. A autenticacao nao e
// exigida porque o URL e um token implicito; o verdadeiro controle de acesso
// e feito pela RLS no momento da criacao do attachment.
func (a *App) handleDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
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
	// enviado por usuario; o cache e seguro porque o nome e unico por upload
	// (trocar a foto gera outra URL).
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("Content-Disposition", "inline; filename="+strconv.Quote(filename))
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
