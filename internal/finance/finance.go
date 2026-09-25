package finance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrCategoryRequired indica que o lancamento veio sem conta do plano de contas.
var ErrCategoryRequired = errors.New("a conta contabil e obrigatoria")

// Category e um item do plano de contas (income | expense).
type Category struct {
	ID        string  `json:"id"`
	BranchID  *string `json:"branch_id,omitempty"`
	Type      string  `json:"type"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	IsActive  bool    `json:"is_active"`
	GroupID   *string `json:"group_id,omitempty"`
	GroupName *string `json:"group_name,omitempty"`
}

// CategoryGroup agrupa contas do plano de contas para relatorios.
type CategoryGroup struct {
	ID        string  `json:"id"`
	BranchID  *string `json:"branch_id,omitempty"`
	Name      string  `json:"name"`
	SortOrder int     `json:"sort_order"`
	IsActive  bool    `json:"is_active"`
}

// Transaction e um lancamento financeiro (append-only).
type Transaction struct {
	ID            string  `json:"id"`
	BranchID      string  `json:"branch_id"`
	CategoryID    *string `json:"category_id,omitempty"`
	CategoryName  *string `json:"category_name,omitempty"`
	AccountID     *string `json:"account_id,omitempty"`
	AccountName   *string `json:"account_name,omitempty"`
	Type          string  `json:"type"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	PaymentMethod *string `json:"payment_method,omitempty"`
	IsAnonymous   bool    `json:"is_anonymous"`
	Description   *string `json:"description,omitempty"`
	ReceiptIssued bool    `json:"receipt_issued"`
	ReceiptID     *string `json:"receipt_id,omitempty"`
	ReceiptRef    *string `json:"receipt_ref,omitempty"`
	ReceiptToken  *string `json:"receipt_token,omitempty"`
	Hash          string  `json:"hash"`
	// OccurredAt e a data efetiva do lancamento (data de competencia, sem hora).
	OccurredAt time.Time `json:"occurred_at"`
	// CreatedAt e o instante em que o registro foi criado (trilha de auditoria).
	CreatedAt time.Time `json:"created_at"`
	// Estorno (o lancamento permanece no historico, mas sai dos relatorios).
	VoidedAt   *time.Time `json:"voided_at,omitempty"`
	VoidReason *string    `json:"void_reason,omitempty"`
	// Anexos (comprovantes) - contagem e URL do primeiro para acesso rapido.
	AttachmentCount int     `json:"attachment_count"`
	AttachmentURL   *string `json:"attachment_url,omitempty"`
	// Fornecedor associado (normalmente em despesas).
	SupplierID   *string `json:"supplier_id,omitempty"`
	SupplierName *string `json:"supplier_name,omitempty"`
}

// MarshalJSON serializa occurred_at como data pura (YYYY-MM-DD). A data efetiva
// e uma data de calendario: devolver o instante completo fazia o front (parseISO
// no fuso local) recuar um dia quando o valor era meia-noite UTC.
func (t Transaction) MarshalJSON() ([]byte, error) {
	type plain Transaction
	return json.Marshal(struct {
		plain
		OccurredAt string `json:"occurred_at"`
	}{
		plain:      plain(t),
		OccurredAt: t.OccurredAt.UTC().Format(dateLayout),
	})
}

type BalanceItem struct {
	CategoryID string  `json:"category_id"`
	Category   string  `json:"category"`
	Type       string  `json:"type"`
	Total      float64 `json:"total"`
}

type Balance struct {
	Income     float64       `json:"income"`
	Expense    float64       `json:"expense"`
	Net        float64       `json:"net"`
	ByCategory []BalanceItem `json:"by_category"`
}

type Repo struct{}

// ListCategories retorna o plano de contas do escopo.
func (r *Repo) ListCategories(ctx context.Context, tx pgx.Tx) ([]Category, error) {
	rows, err := tx.Query(ctx, `
		SELECT c.id::text, COALESCE(c.branch_id::text,''), c.type, c.code, c.name, c.is_active,
		       c.group_id::text, g.name
		FROM financial_categories c
		LEFT JOIN financial_category_groups g ON g.id = c.group_id
		ORDER BY c.type, c.code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		var bid string
		if err := rows.Scan(&c.ID, &bid, &c.Type, &c.Code, &c.Name, &c.IsActive,
			&c.GroupID, &c.GroupName); err != nil {
			return nil, err
		}
		if bid != "" {
			c.BranchID = &bid
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type CreateCategoryInput struct {
	Type    string  `json:"type"`
	Code    string  `json:"code"`
	Name    string  `json:"name"`
	GroupID *string `json:"group_id"`
}

func (r *Repo) CreateCategory(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateCategoryInput) (*Category, error) {
	var c Category
	err := tx.QueryRow(ctx, `
		INSERT INTO financial_categories (tenant_id, branch_id, type, code, name, group_id)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5, NULLIF($6,'')::uuid)
		RETURNING id::text, COALESCE(branch_id::text,''), type, code, name, is_active, group_id::text,
		  (SELECT g.name FROM financial_category_groups g WHERE g.id = group_id)`,
		tenantID, branchID, in.Type, in.Code, in.Name, in.GroupID).
		Scan(&c.ID, &c.BranchID, &c.Type, &c.Code, &c.Name, &c.IsActive, &c.GroupID, &c.GroupName)
	return &c, err
}

// UpdateCategoryInput e o payload de edicao do plano de contas (PATCH: nil mantem).
type UpdateCategoryInput struct {
	Type     *string `json:"type"`
	Code     *string `json:"code"`
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
	GroupID  *string `json:"group_id"`
}

// ErrCategoryInUse indica que a conta contabil tem lancamentos/subitens.
var ErrCategoryInUse = errors.New("conta contabil em uso: desative em vez de excluir")

// UpdateCategory edita codigo/nome/tipo/grupo e ativa/desativa uma conta contabil.
// group_id aceita "" para DESVINCULAR (NULL).
func (r *Repo) UpdateCategory(ctx context.Context, tx pgx.Tx, id string, in UpdateCategoryInput) (*Category, error) {
	var c Category
	err := tx.QueryRow(ctx, `
		UPDATE financial_categories SET
			type = COALESCE(NULLIF($2,''), type),
			code = COALESCE(NULLIF($3,''), code),
			name = COALESCE(NULLIF($4,''), name),
			is_active = COALESCE($5::boolean, is_active),
			group_id = CASE WHEN $6::text IS NULL THEN group_id
			                WHEN btrim($6::text) = '' THEN NULL
			                ELSE $6::uuid END
		WHERE id = $1::uuid
		RETURNING id::text, COALESCE(branch_id::text,''), type, code, name, is_active, group_id::text,
		  (SELECT g.name FROM financial_category_groups g WHERE g.id = group_id)`,
		id, in.Type, in.Code, in.Name, in.IsActive, in.GroupID).
		Scan(&c.ID, &c.BranchID, &c.Type, &c.Code, &c.Name, &c.IsActive, &c.GroupID, &c.GroupName)
	return &c, err
}

// ---- Grupos de contas ----

func (r *Repo) ListCategoryGroups(ctx context.Context, tx pgx.Tx) ([]CategoryGroup, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, COALESCE(branch_id::text,''), name, sort_order, is_active
		FROM financial_category_groups ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CategoryGroup{}
	for rows.Next() {
		var g CategoryGroup
		var bid string
		if err := rows.Scan(&g.ID, &bid, &g.Name, &g.SortOrder, &g.IsActive); err != nil {
			return nil, err
		}
		if bid != "" {
			g.BranchID = &bid
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

type CreateCategoryGroupInput struct {
	Name      string `json:"name"`
	SortOrder *int   `json:"sort_order"`
}

func (r *Repo) CreateCategoryGroup(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateCategoryGroupInput) (*CategoryGroup, error) {
	order := 0
	if in.SortOrder != nil {
		order = *in.SortOrder
	}
	var g CategoryGroup
	err := tx.QueryRow(ctx, `
		INSERT INTO financial_category_groups (tenant_id, branch_id, name, sort_order)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4)
		RETURNING id::text, COALESCE(branch_id::text,''), name, sort_order, is_active`,
		tenantID, branchID, in.Name, order).
		Scan(&g.ID, &g.BranchID, &g.Name, &g.SortOrder, &g.IsActive)
	return &g, err
}

type UpdateCategoryGroupInput struct {
	Name      *string `json:"name"`
	SortOrder *int    `json:"sort_order"`
	IsActive  *bool   `json:"is_active"`
}

func (r *Repo) UpdateCategoryGroup(ctx context.Context, tx pgx.Tx, id string, in UpdateCategoryGroupInput) (*CategoryGroup, error) {
	var g CategoryGroup
	err := tx.QueryRow(ctx, `
		UPDATE financial_category_groups SET
			name = COALESCE(NULLIF($2,''), name),
			sort_order = COALESCE($3::int, sort_order),
			is_active = COALESCE($4::boolean, is_active)
		WHERE id = $1::uuid
		RETURNING id::text, COALESCE(branch_id::text,''), name, sort_order, is_active`,
		id, in.Name, in.SortOrder, in.IsActive).
		Scan(&g.ID, &g.BranchID, &g.Name, &g.SortOrder, &g.IsActive)
	return &g, err
}

func (r *Repo) DeleteCategoryGroup(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM financial_category_groups WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteCategory exclui uma conta contabil sem uso (senao devolve ErrCategoryInUse).
func (r *Repo) DeleteCategory(ctx context.Context, tx pgx.Tx, id string) error {
	var used int
	if err := tx.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM financial_transactions WHERE category_id = $1::uuid)
		     + (SELECT count(*) FROM financial_categories WHERE parent_id = $1::uuid)`, id).Scan(&used); err != nil {
		return err
	}
	if used > 0 {
		return ErrCategoryInUse
	}
	tag, err := tx.Exec(ctx, `DELETE FROM financial_categories WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

type CreateTxnInput struct {
	Type          string  `json:"type"`
	Amount        float64 `json:"amount"`
	CategoryID    *string `json:"category_id"`
	AccountID     *string `json:"account_id"`
	PaymentMethod *string `json:"payment_method"`
	IsAnonymous   bool    `json:"is_anonymous"`
	Description   *string `json:"description"`
	DonorMemberID *string `json:"donor_member_id"`
	BenefactorID  *string `json:"benefactor_id"`
	SupplierID    *string `json:"supplier_id"`
	OccurredAt    *string `json:"occurred_at"`
	// Rateio opcional do lancamento entre eventos (custo real por evento).
	EventAllocations []EventAllocationInput `json:"event_allocations"`
}

type EventAllocationInput struct {
	EventID string   `json:"event_id"`
	Amount  *float64 `json:"amount"` // nulo => divide o restante igualmente
}

// EventAllocation e o rateio de um lancamento em um evento.
type EventAllocation struct {
	ID        string  `json:"id"`
	EventID   string  `json:"event_id"`
	EventName string  `json:"event_name"`
	Amount    float64 `json:"amount"`
}

// Create insere um lancamento e retorna a transacao + id, ref e token do recibo gerado.
// A conta do plano de contas (category_id) e OBRIGATORIA e precisa bater com o
// tipo do lancamento (nao da para lancar uma despesa numa conta de entrada).
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateTxnInput) (*Transaction, string, string, string, error) {
	if in.CategoryID == nil || *in.CategoryID == "" {
		return nil, "", "", "", ErrCategoryRequired
	}
	// O RLS esconde contas fora do escopo, entao um id de outro tenant/filial
	// resulta em ErrNoRows - tratado como conta invalida.
	var catType string
	if err := tx.QueryRow(ctx, `SELECT type FROM financial_categories WHERE id = $1::uuid`, *in.CategoryID).Scan(&catType); err != nil {
		return nil, "", "", "", err
	}
	if catType != in.Type {
		return nil, "", "", "", fmt.Errorf("a conta selecionada e de %s, mas o lancamento e de %s", catType, in.Type)
	}

	// A data efetiva vem do formulario (AAAA-MM-DD, DD/MM/AAAA ou RFC3339). Antes
	// so RFC3339 era aceito e, ao falhar, caia silenciosamente em "agora" - por
	// isso todo lancamento retroativo virava a data de hoje.
	when, err := parseOccurredAt(in.OccurredAt)
	if err != nil {
		return nil, "", "", "", err
	}
	status := false
	var t Transaction
	err = tx.QueryRow(ctx, `
		INSERT INTO financial_transactions
			(tenant_id, branch_id, category_id, type, amount, currency,
			 payment_method, donor_member_id, benefactor_id, supplier_id, is_anonymous, description, occurred_at, account_id)
		VALUES ($1, NULLIF($2,'')::uuid, NULLIF($3,'')::uuid, $4, $5, 'BRL',
			 $6, NULLIF($7,'')::uuid, NULLIF($8,'')::uuid, NULLIF($13,'')::uuid, $9, $10, $11, NULLIF($12,'')::uuid)
		RETURNING id::text, branch_id::text, category_id::text, account_id::text,
		          type, amount::float8, currency, payment_method, is_anonymous,
		          description, receipt_issued, hash, occurred_at, created_at`,
		tenantID, branchID, nullIfEmpty(in.CategoryID), in.Type, in.Amount,
		in.PaymentMethod, in.DonorMemberID, in.BenefactorID, in.IsAnonymous, in.Description, when, in.AccountID,
		in.SupplierID).
		Scan(&t.ID, &t.BranchID, &t.CategoryID, &t.AccountID, &t.Type, &t.Amount, &t.Currency,
			&t.PaymentMethod, &t.IsAnonymous, &t.Description, &status, &t.Hash, &t.OccurredAt, &t.CreatedAt)
	if err != nil {
		return nil, "", "", "", err
	}
	t.ReceiptIssued = status

	// Gera recibo digital automatico (documento).
	docID, ref, token, err := r.issueReceipt(ctx, tx, tenantID, branchID, t)
	if err != nil {
		return nil, "", "", "", err
	}
	// Rateio entre eventos (opcional).
	if err := r.insertEventAllocations(ctx, tx, tenantID, branchID, t.ID, in.Amount, in.EventAllocations); err != nil {
		return nil, "", "", "", err
	}
	return &t, docID, ref, token, nil
}

// insertEventAllocations grava o rateio do lancamento entre eventos. Valores nao
// informados dividem o restante igualmente; eventos fora do escopo sao recusados.
func (r *Repo) insertEventAllocations(ctx context.Context, tx pgx.Tx, tenantID, branchID, txID string, total float64, allocs []EventAllocationInput) error {
	if len(allocs) == 0 {
		return nil
	}
	var fixed float64
	missing := 0
	for _, a := range allocs {
		if a.Amount != nil {
			fixed += *a.Amount
		} else {
			missing++
		}
	}
	perMissing := 0.0
	if missing > 0 {
		perMissing = (total - fixed) / float64(missing)
		if perMissing < 0 {
			perMissing = 0
		}
	}
	for _, a := range allocs {
		amt := perMissing
		if a.Amount != nil {
			amt = *a.Amount
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO financial_event_allocations (tenant_id, branch_id, transaction_id, event_id, amount)
			SELECT $1, $2::uuid, $3::uuid, e.id, $5
			FROM church_events e
			WHERE e.id = $4::uuid AND e.tenant_id = $1::uuid AND e.branch_id = $2::uuid`,
			tenantID, branchID, txID, a.EventID, amt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("evento %s nao encontrado no escopo", a.EventID)
		}
	}
	return nil
}

// ListTransactionEvents retorna o rateio de um lancamento por evento.
func (r *Repo) ListTransactionEvents(ctx context.Context, tx pgx.Tx, txID string) ([]EventAllocation, error) {
	rows, err := tx.Query(ctx, `
		SELECT a.id::text, a.event_id::text,
		       COALESCE(to_char(e.starts_at, 'DD/MM/YYYY') || ' - ' || k.name, 'Evento'),
		       a.amount::float8
		FROM financial_event_allocations a
		LEFT JOIN church_events e ON e.id = a.event_id
		LEFT JOIN event_kinds k ON k.id = e.kind_id
		WHERE a.transaction_id = $1::uuid
		ORDER BY a.created_at`, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EventAllocation{}
	for rows.Next() {
		var a EventAllocation
		if err := rows.Scan(&a.ID, &a.EventID, &a.EventName, &a.Amount); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Void anula um lancamento (estorno). O registro permanece no historico; os
// relatorios param de conta-lo. Nao e DELETE nem altera os valores.
func (r *Repo) Void(ctx context.Context, tx pgx.Tx, id, reason, actorID string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE financial_transactions
		SET voided_at = now(), voided_by = NULLIF($2,'')::uuid, void_reason = NULLIF($3,'')
		WHERE id = $1::uuid AND voided_at IS NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM financial_audit_items fa
		      JOIN financial_audits a ON a.id = fa.audit_id
		      WHERE fa.transaction_id = financial_transactions.id AND a.status = 'fechada')`,
		id, actorID, reason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Delete remove DEFINITIVAMENTE um lancamento e recalcula a hash-chain do
// tenant, para a trilha nao ficar com lacunas. Um lancamento preso a auditoria
// fechada nao pode ser excluido (mesma trava do estorno).
func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	var tenantID string
	err := tx.QueryRow(ctx, `
		DELETE FROM financial_transactions
		WHERE id = $1::uuid
		  AND NOT EXISTS (
		      SELECT 1 FROM financial_audit_items fa
		      JOIN financial_audits a ON a.id = fa.audit_id
		      WHERE fa.transaction_id = financial_transactions.id AND a.status = 'fechada')
		RETURNING tenant_id::text`, id).Scan(&tenantID)
	if err != nil {
		return err
	}
	// Remove o recibo digital (documento) vinculado ao lancamento.
	if _, err := tx.Exec(ctx,
		`DELETE FROM documents WHERE kind = 'receipt' AND content->>'tx_id' = $1`, id); err != nil {
		return err
	}
	// Recalcula a cadeia do tenant (RPC SECURITY DEFINER - ver migracao 000061).
	if _, err := tx.Exec(ctx, `SELECT fin_tx_rechain($1::uuid)`, tenantID); err != nil {
		return err
	}
	return nil
}

// CreateBatch insere varios lancamentos de uma vez (entrada rapida). Cada linha
// roda em um SAVEPOINT: uma linha invalida nao derruba as demais e o resultado
// traz o motivo por linha (mesma forma do ImportResult).
func (r *Repo) CreateBatch(ctx context.Context, tx pgx.Tx, tenantID, branchID string, inputs []CreateTxnInput) (ImportResult, error) {
	res := ImportResult{Errors: []ImportError{}, CreatedIDs: make([]string, len(inputs))}
	for i, in := range inputs {
		sp := "finb_" + strconv.Itoa(i)
		if _, err := tx.Exec(ctx, "SAVEPOINT "+sp); err != nil {
			return res, err
		}
		created, _, _, _, err := r.Create(ctx, tx, tenantID, branchID, in)
		if err != nil {
			if _, rbErr := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+sp); rbErr != nil {
				return res, rbErr
			}
			res.Skipped++
			res.Errors = append(res.Errors, ImportError{Line: i + 1, Error: err.Error()})
			continue
		}
		if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT "+sp); err != nil {
			return res, err
		}
		res.CreatedIDs[i] = created.ID
		res.Imported++
	}
	return res, nil
}

// List retorna lancamentos do escopo, com filtro opcional de tipo.
func (r *Repo) List(ctx context.Context, tx pgx.Tx, kind string) ([]Transaction, error) {
	rows, err := tx.Query(ctx, `
		WITH recent AS (
		    SELECT id
		    FROM financial_transactions
		    WHERE ($1 = '' OR type = $1)
		    ORDER BY occurred_at DESC, created_at DESC, id DESC
		)
		SELECT t.id::text, t.branch_id::text, t.category_id::text, c.name,
		       t.account_id::text, a.name,
		       t.supplier_id::text, s.name,
		       t.type, t.amount::float8, t.currency, t.payment_method,
		       t.is_anonymous, t.description, t.receipt_issued, t.hash, t.occurred_at, t.created_at,
		       d.id::text, d.document_ref, d.qr_token,
		       t.voided_at, t.void_reason,
		       (SELECT count(*) FROM financial_attachments fa WHERE fa.transaction_id = t.id)::int,
		       (SELECT fa.file_url FROM financial_attachments fa WHERE fa.transaction_id = t.id ORDER BY fa.created_at LIMIT 1)
		FROM financial_transactions t
		JOIN recent r ON r.id = t.id
		LEFT JOIN financial_categories c ON c.id = t.category_id
		LEFT JOIN financial_accounts a ON a.id = t.account_id
		LEFT JOIN suppliers s ON s.id = t.supplier_id
		LEFT JOIN documents d ON d.kind = 'receipt' AND d.content->>'tx_id' = t.id::text
		ORDER BY t.occurred_at, t.created_at, t.id`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.BranchID, &t.CategoryID, &t.CategoryName,
			&t.AccountID, &t.AccountName,
			&t.SupplierID, &t.SupplierName,
			&t.Type, &t.Amount, &t.Currency, &t.PaymentMethod,
			&t.IsAnonymous, &t.Description, &t.ReceiptIssued, &t.Hash, &t.OccurredAt, &t.CreatedAt,
			&t.ReceiptID, &t.ReceiptRef, &t.ReceiptToken,
			&t.VoidedAt, &t.VoidReason, &t.AttachmentCount, &t.AttachmentURL); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SumBalance retorna entradas/saidas agregadas e por categoria (balancete/DRE).
// O saldo liquido inclui o initial_balance das contas bancarias visiveis no
// escopo, garantindo que o saldo reflit a realidade bancaria mesmo quando nao
// ha lancamentos registrados para uma conta (ex.: saldo inicial pre-cadastro).
func (r *Repo) SumBalance(ctx context.Context, tx pgx.Tx, kind string) (Balance, error) {
	var b Balance
	var initialBalance float64
	err := tx.QueryRow(ctx, `
		WITH tx_agg AS (
			SELECT COALESCE(SUM(amount) FILTER (WHERE type='income'),0) AS inc,
			       COALESCE(SUM(amount) FILTER (WHERE type='expense'),0) AS exp
			FROM financial_transactions
			WHERE ($1 = '' OR type = $1)
			  AND voided_at IS NULL
		),
		acct_init AS (
			SELECT COALESCE(SUM(initial_balance),0) AS init_balance
			FROM financial_accounts
			WHERE is_active
		)
		SELECT tx_agg.inc, tx_agg.exp, acct_init.init_balance
		FROM tx_agg CROSS JOIN acct_init`, kind).
		Scan(&b.Income, &b.Expense, &initialBalance)
	if err != nil {
		return b, err
	}
	b.Net = b.Income - b.Expense + initialBalance

	rows, err := tx.Query(ctx, `
		SELECT COALESCE(t.category_id::text,''), COALESCE(c.name,'Sem categoria'), t.type, SUM(t.amount)::float8
		FROM financial_transactions t
		LEFT JOIN financial_categories c ON c.id = t.category_id
		WHERE ($1 = '' OR t.type = $1)
		  AND t.voided_at IS NULL
		GROUP BY 1,2,3 ORDER BY 4 DESC`, kind)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	for rows.Next() {
		var it BalanceItem
		if err := rows.Scan(&it.CategoryID, &it.Category, &it.Type, &it.Total); err != nil {
			return b, err
		}
		b.ByCategory = append(b.ByCategory, it)
	}
	return b, rows.Err()
}

func nullIfEmpty(s *string) *string {
	if s != nil && *s == "" {
		return nil
	}
	return s
}
