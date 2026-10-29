package finance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrCategoryRequired indica que o lancamento veio sem conta do plano de contas.
var ErrCategoryRequired = errors.New("a conta contabil e obrigatoria")

// Erros de validacao do lancamento (conta bancaria, valor e travas).
var (
	ErrAccountRequired       = errors.New("a conta bancaria e obrigatoria")
	ErrAccountNotFound       = errors.New("conta bancaria nao encontrada no escopo")
	ErrAccountInactive       = errors.New("conta bancaria inativa")
	ErrAmountNotPositive     = errors.New("o valor do lancamento deve ser maior que zero")
	ErrTransactionReconciled = errors.New("lancamento conciliado: nao pode ser alterado ou excluido")
	ErrTransactionVoided     = errors.New("lancamento estornado: nao pode ser conciliado")
	ErrPeriodClosed          = errors.New("periodo encerrado: lancamentos nao podem ser criados ou alterados")
	ErrAuditClosed           = errors.New("auditoria fechada: lancamentos nao podem ser alterados")
	// ErrScopeReconciled impede inserir no meio de uma sequencia que tem
	// lancamento conciliado (o UPDATE de entry_seq esbarraria no trigger).
	ErrScopeReconciled = errors.New("ha lancamento conciliado nessa posicao; desconcile antes de inserir")
	// ErrGroupedReconcile indica que a conciliacao deve ser feita pelo grupo.
	ErrGroupedReconcile = errors.New("lancamento agrupado: concilie pelo grupo")
)

// assertTxnNotGrouped impede estornar/excluir um membro de grupo (o vinculo
// precisa ser desfeito antes).
func assertTxnNotGrouped(ctx context.Context, tx pgx.Tx, id string) error {
	var groupID *string
	if err := tx.QueryRow(ctx, `SELECT group_id::text FROM financial_transactions WHERE id = $1::uuid`, id).Scan(&groupID); err != nil {
		return err
	}
	if groupID != nil {
		return ErrTransactionGrouped
	}
	return nil
}

// assertDateOpen confere se a data/filial nao esta dentro de uma conciliacao
// encerrada ou auditoria fechada. A checagem no banco continua sendo a
// autoridade final (inclusive para concorrencia).
func assertDateOpen(ctx context.Context, tx pgx.Tx, tenantID, branchID, date string) error {
	var closed bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM financial_reconciliations fr
			WHERE fr.tenant_id = $1::uuid
			  AND fr.branch_id = $2::uuid
			  AND fr.status = 'conciliada'
			  AND $3::date BETWEEN fr.period_start AND fr.period_end
			UNION ALL
			SELECT 1 FROM financial_audits a
			WHERE a.tenant_id = $1::uuid
			  AND a.branch_id = $2::uuid
			  AND a.status = 'fechada'
			  AND $3::date BETWEEN a.period_start AND a.period_end
		)`, tenantID, branchID, date).Scan(&closed); err != nil {
		return err
	}
	if closed {
		return ErrPeriodClosed
	}
	return nil
}

// assertTxnMutable bloqueia alteracoes em lancamento conciliado, em periodo
// encerrado ou preso a auditoria fechada.
func assertTxnMutable(ctx context.Context, tx pgx.Tx, id string) error {
	var tenantID, branchID, date string
	var reconciledAt *time.Time
	if err := tx.QueryRow(ctx, `
		SELECT tenant_id::text, branch_id::text,
		       (occurred_at AT TIME ZONE 'UTC')::date::text,
		       reconciled_at
		FROM financial_transactions WHERE id = $1::uuid`, id).
		Scan(&tenantID, &branchID, &date, &reconciledAt); err != nil {
		return err
	}
	if reconciledAt != nil {
		return ErrTransactionReconciled
	}
	if err := assertDateOpen(ctx, tx, tenantID, branchID, date); err != nil {
		return err
	}
	var auditClosed bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM financial_audit_items fa
			JOIN financial_audits a ON a.id = fa.audit_id
			WHERE fa.transaction_id = $1::uuid AND a.status = 'fechada')`, id).Scan(&auditClosed); err != nil {
		return err
	}
	if auditClosed {
		return ErrAuditClosed
	}
	return nil
}

// validateTxnAccount confere a conta bancaria do lancamento. currentAccountID
// permite manter uma conta inativa que ja estava vinculada ao registro (edicao).
func validateTxnAccount(ctx context.Context, tx pgx.Tx, accountID, currentAccountID string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return ErrAccountRequired
	}
	var active bool
	if err := tx.QueryRow(ctx, `SELECT is_active FROM financial_accounts WHERE id = $1::uuid`, accountID).Scan(&active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		return err
	}
	if !active && accountID != strings.TrimSpace(currentAccountID) {
		return ErrAccountInactive
	}
	return nil
}

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
	// EntrySeq e a sequencia manual do lancamento no escopo
	// (tenant, filial, conta bancaria, data). Serve para alinhar a ordem com o
	// extrato. Fica FORA da hash-chain e pode ser alterada (dado de ordenacao).
	EntrySeq *int `json:"entry_seq,omitempty"`
	// Conciliacao individual (nao depende de um periodo).
	Reconciled   bool       `json:"reconciled"`
	ReconciledAt *time.Time `json:"reconciled_at,omitempty"`
	// Estorno (o lancamento permanece no historico, mas sai dos relatorios).
	VoidedAt   *time.Time `json:"voided_at,omitempty"`
	VoidReason *string    `json:"void_reason,omitempty"`
	// Anexos (comprovantes) - contagem e URL do primeiro para acesso rapido.
	AttachmentCount int     `json:"attachment_count"`
	AttachmentURL   *string `json:"attachment_url,omitempty"`
	// Fornecedor associado (normalmente em despesas).
	SupplierID   *string `json:"supplier_id,omitempty"`
	SupplierName *string `json:"supplier_name,omitempty"`
	// Origem da entrada: membro doador, benfeitor cadastrado ou identificacao
	// livre (doador avulso). DonorName e resolvido (donor_name > membro > benfeitor).
	DonorMemberID *string `json:"donor_member_id,omitempty"`
	BenefactorID  *string `json:"benefactor_id,omitempty"`
	DonorName     *string `json:"donor_name,omitempty"`
	// Contas a pagar: vencimento, quitacao (NULL = em aberto) e centro de custo.
	DueDate    *string    `json:"due_date,omitempty"`
	PaidAt     *time.Time `json:"paid_at,omitempty"`
	CostCenter *string    `json:"cost_center,omitempty"`
	// Agrupamento bancario: quando preenchido, o lancamento compoe um grupo
	// (uma unica linha no extrato). GroupStatus e "agrupado" ou "conciliado".
	GroupID     *string `json:"group_id,omitempty"`
	GroupStatus *string `json:"group_status,omitempty"`
	// Transferencia entre contas: as duas pernas (saida/entrada) compartilham o
	// mesmo id. TransferKind e deposit|redeem|transfer e CounterpartAccountName
	// e a outra conta envolvida (para exibicao).
	AccountTransferID      *string `json:"account_transfer_id,omitempty"`
	TransferKind           *string `json:"transfer_kind,omitempty"`
	CounterpartAccountName *string `json:"counterpart_account_name,omitempty"`
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
	Code       string  `json:"code"`
	Category   string  `json:"category"`
	Type       string  `json:"type"`
	Total      float64 `json:"total"`
}

// AccountTypeBalance consolida o saldo das contas de um mesmo tipo.
type AccountTypeBalance struct {
	AccountType    string  `json:"account_type"`
	InitialBalance float64 `json:"initial_balance"`
	Income         float64 `json:"income"`
	Expense        float64 `json:"expense"`
	Net            float64 `json:"net"`
}

type Balance struct {
	Income        float64              `json:"income"`
	Expense       float64              `json:"expense"`
	Net           float64              `json:"net"`
	ByCategory    []BalanceItem        `json:"by_category"`
	ByAccountType []AccountTypeBalance `json:"by_account_type"`
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
	DonorName     *string `json:"donor_name"`
	SupplierID    *string `json:"supplier_id"`
	OccurredAt    *string `json:"occurred_at"`
	// BranchID permite que a Sede grave em uma filial especifica (ex.: inserir
	// um lancamento entre linhas de outra filial no escopo "Todas"). Ignorado
	// para usuarios de filial e validado pelo handler.
	BranchID *string `json:"branch_id"`
	// EntrySeq e a posicao desejada no escopo (tenant, filial, conta, data).
	// Se informada, empurra os lancamentos daquela posicao em diante; se nula,
	// o lancamento e anexado ao fim do escopo.
	EntrySeq *int `json:"entry_seq"`
	// Rateio opcional do lancamento entre eventos (custo real por evento).
	EventAllocations []EventAllocationInput `json:"event_allocations"`
	// Contas a pagar (opcional): vencimento, centro de custo e status pago.
	DueDate    *string `json:"due_date"`
	CostCenter *string `json:"cost_center"`
	Paid       *bool   `json:"paid"`
	// Reconciled cria o lancamento ja conciliado (lançar como conciliado).
	// ActorID e preenchido pelo handler, nunca pelo cliente.
	Reconciled bool   `json:"reconciled"`
	ActorID    string `json:"-"`
	// AccountTransferID marca uma perna de transferencia entre contas. Nesse
	// caso a categoria e opcional, nao ha recibo nem split de repasses.
	AccountTransferID *string `json:"-"`
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
	isTransfer := in.AccountTransferID != nil && *in.AccountTransferID != ""
	if !isTransfer && (in.CategoryID == nil || *in.CategoryID == "") {
		return nil, "", "", "", ErrCategoryRequired
	}
	if in.Amount <= 0 {
		return nil, "", "", "", ErrAmountNotPositive
	}
	acct := ""
	if in.AccountID != nil {
		acct = *in.AccountID
	}
	if err := validateTxnAccount(ctx, tx, acct, ""); err != nil {
		return nil, "", "", "", err
	}
	// O RLS esconde contas fora do escopo, entao um id de outro tenant/filial
	// resulta em ErrNoRows - tratado como conta invalida.
	if !isTransfer {
		var catType string
		if err := tx.QueryRow(ctx, `SELECT type FROM financial_categories WHERE id = $1::uuid`, *in.CategoryID).Scan(&catType); err != nil {
			return nil, "", "", "", err
		}
		if catType != in.Type {
			return nil, "", "", "", fmt.Errorf("a conta selecionada e de %s, mas o lancamento e de %s", catType, in.Type)
		}
	}

	// A identificacao livre (doador avulso) so vale quando nao ha membro nem
	// benfeitor vinculado - evita gravar o nome resolvido de volta em donor_name
	// ao editar um lancamento identificado.
	if (in.DonorMemberID != nil && *in.DonorMemberID != "") ||
		(in.BenefactorID != nil && *in.BenefactorID != "") {
		in.DonorName = nil
	}

	// A data efetiva vem do formulario (AAAA-MM-DD, DD/MM/AAAA ou RFC3339). Antes
	// so RFC3339 era aceito e, ao falhar, caia silenciosamente em "agora" - por
	// isso todo lancamento retroativo virava a data de hoje.
	when, err := parseOccurredAt(in.OccurredAt)
	if err != nil {
		return nil, "", "", "", err
	}

	// Periodo encerrado nao aceita lancamento novo (inclusive retroativo).
	scopeDate := when.UTC().Format(dateLayout)
	if err := assertDateOpen(ctx, tx, tenantID, branchID, scopeDate); err != nil {
		return nil, "", "", "", err
	}
	// Sequencia do lancamento no escopo (tenant, filial, conta, data). O lock
	// por escopo evita corrida ao calcular a proxima posicao / renumerar.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		tenantID+"|"+branchID+"|"+acct+"|"+scopeDate); err != nil {
		return nil, "", "", "", err
	}
	entrySeq := 0
	if in.EntrySeq != nil && *in.EntrySeq > 0 {
		entrySeq = *in.EntrySeq
		// A insercao no meio empurra os lancamentos da posicao em diante. Se
		// algum deles estiver conciliado, o trigger bloquearia o UPDATE com um
		// erro generico - devolvemos um erro claro antes.
		var locked bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM financial_transactions
				WHERE tenant_id = $1::uuid
				  AND branch_id = NULLIF($2,'')::uuid
				  AND COALESCE(account_id, '00000000-0000-0000-0000-000000000000'::uuid)
				      = COALESCE(NULLIF($3,'')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
				  AND (occurred_at AT TIME ZONE 'UTC')::date = $4::date
				  AND entry_seq >= $5
				  AND reconciled_at IS NOT NULL)`,
			tenantID, branchID, acct, scopeDate, entrySeq).Scan(&locked); err != nil {
			return nil, "", "", "", err
		}
		if locked {
			return nil, "", "", "", ErrScopeReconciled
		}
		// Grupos tambem ocupam uma posicao na sequencia; a insercao precisa
		// empurra-los junto com os lancamentos.
		var groupLocked bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM financial_transaction_groups
				WHERE tenant_id = $1::uuid
				  AND branch_id = NULLIF($2,'')::uuid
				  AND account_id = NULLIF($3,'')::uuid
				  AND occurred_at = $4::date
				  AND entry_seq >= $5
				  AND status = 'conciliado')`,
			tenantID, branchID, acct, scopeDate, entrySeq).Scan(&groupLocked); err != nil {
			return nil, "", "", "", err
		}
		if groupLocked {
			return nil, "", "", "", ErrScopeReconciled
		}
		if _, err := tx.Exec(ctx, `
			UPDATE financial_transaction_groups
			SET entry_seq = entry_seq + 1
			WHERE tenant_id = $1::uuid
			  AND branch_id = NULLIF($2,'')::uuid
			  AND account_id = NULLIF($3,'')::uuid
			  AND occurred_at = $4::date
			  AND entry_seq >= $5
			  AND status = 'agrupado'`,
			tenantID, branchID, acct, scopeDate, entrySeq); err != nil {
			return nil, "", "", "", err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE financial_transactions
			SET entry_seq = entry_seq + 1
			WHERE tenant_id = $1::uuid
			  AND branch_id = NULLIF($2,'')::uuid
			  AND COALESCE(account_id, '00000000-0000-0000-0000-000000000000'::uuid)
			      = COALESCE(NULLIF($3,'')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
			  AND (occurred_at AT TIME ZONE 'UTC')::date = $4::date
			  AND entry_seq >= $5`,
			tenantID, branchID, acct, scopeDate, entrySeq); err != nil {
			return nil, "", "", "", err
		}
	} else {
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(MAX(seq), 0) + 1
			FROM (
			    SELECT entry_seq AS seq
			      FROM financial_transactions
			     WHERE tenant_id = $1::uuid
			       AND branch_id = NULLIF($2,'')::uuid
			       AND COALESCE(account_id, '00000000-0000-0000-0000-000000000000'::uuid)
			           = COALESCE(NULLIF($3,'')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
			       AND (occurred_at AT TIME ZONE 'UTC')::date = $4::date
			       AND group_id IS NULL
			    UNION ALL
			    SELECT entry_seq AS seq
			      FROM financial_transaction_groups
			     WHERE tenant_id = $1::uuid
			       AND branch_id = NULLIF($2,'')::uuid
			       AND account_id = NULLIF($3,'')::uuid
			       AND occurred_at = $4::date
			) q`,
			tenantID, branchID, acct, scopeDate).Scan(&entrySeq); err != nil {
			return nil, "", "", "", err
		}
	}

	status := false
	var outSeq int
	var t Transaction
	err = tx.QueryRow(ctx, `
		INSERT INTO financial_transactions
			(tenant_id, branch_id, category_id, type, amount, currency,
			 payment_method, donor_member_id, benefactor_id, donor_name, supplier_id, is_anonymous, description, occurred_at, account_id, entry_seq,
			 due_date, cost_center, paid_at, reconciled_at, reconciled_by, account_transfer_id)
		VALUES ($1, NULLIF($2,'')::uuid, NULLIF($3,'')::uuid, $4, $5, 'BRL',
			 $6, NULLIF($7,'')::uuid, NULLIF($8,'')::uuid, NULLIF($14,''), NULLIF($13,'')::uuid, $9, $10, $11, NULLIF($12,'')::uuid, $15,
			 NULLIF($16,'')::date, NULLIF($17,''), CASE WHEN $18 THEN now() ELSE NULL END,
			 CASE WHEN $19 THEN now() ELSE NULL END,
			 CASE WHEN $19 THEN NULLIF($20,'')::uuid ELSE NULL END,
			 NULLIF($21,'')::uuid)
		RETURNING id::text, branch_id::text, category_id::text, account_id::text,
		          donor_member_id::text, benefactor_id::text, donor_name, entry_seq,
		          type, amount::float8, currency, payment_method, is_anonymous,
		          description, receipt_issued, hash, occurred_at, created_at,
		          due_date::text, paid_at, cost_center, reconciled_at, account_transfer_id::text`,
		tenantID, branchID, nullIfEmpty(in.CategoryID), in.Type, in.Amount,
		in.PaymentMethod, in.DonorMemberID, in.BenefactorID, in.IsAnonymous, in.Description, when, in.AccountID,
		in.SupplierID, in.DonorName, entrySeq, in.DueDate, in.CostCenter, in.Paid != nil && *in.Paid,
		in.Reconciled, in.ActorID, in.AccountTransferID).
		Scan(&t.ID, &t.BranchID, &t.CategoryID, &t.AccountID,
			&t.DonorMemberID, &t.BenefactorID, &t.DonorName, &outSeq,
			&t.Type, &t.Amount, &t.Currency,
			&t.PaymentMethod, &t.IsAnonymous, &t.Description, &status, &t.Hash, &t.OccurredAt, &t.CreatedAt,
			&t.DueDate, &t.PaidAt, &t.CostCenter, &t.ReconciledAt, &t.AccountTransferID)
	if err != nil {
		return nil, "", "", "", err
	}
	t.EntrySeq = &outSeq
	t.ReceiptIssued = status
	t.Reconciled = t.ReconciledAt != nil

	// Perna de transferencia interna: nao emite recibo, nao tem rateio de evento
	// e nao entra no split de repasses (nao e receita/despesa de resultado).
	if isTransfer {
		return &t, "", "", "", nil
	}

	// Gera recibo digital automatico (documento).
	docID, ref, token, err := r.issueReceipt(ctx, tx, tenantID, branchID, t)
	if err != nil {
		return nil, "", "", "", err
	}
	// Rateio entre eventos (opcional).
	if err := r.insertEventAllocations(ctx, tx, tenantID, branchID, t.ID, in.EventAllocations); err != nil {
		return nil, "", "", "", err
	}
	// Split automatico de repasses (somente ENTRADAS, quando ligado).
	if t.Type == "income" {
		if err := r.ApplySplit(ctx, tx, tenantID, t.BranchID, t.ID, t.Amount); err != nil {
			return nil, "", "", "", err
		}
	}
	return &t, docID, ref, token, nil
}

// UpdateTxnInput e o corpo da edicao de um lancamento (PATCH). Nao inclui
// entry_seq: a posicao do lancamento NAO muda ao editar (reposicionar e feito
// por POST /finance/transactions/reorder).
type UpdateTxnInput struct {
	Type          string  `json:"type"`
	Amount        float64 `json:"amount"`
	CategoryID    *string `json:"category_id"`
	AccountID     *string `json:"account_id"`
	PaymentMethod *string `json:"payment_method"`
	IsAnonymous   bool    `json:"is_anonymous"`
	Description   *string `json:"description"`
	DonorMemberID *string `json:"donor_member_id"`
	BenefactorID  *string `json:"benefactor_id"`
	DonorName     *string `json:"donor_name"`
	SupplierID    *string `json:"supplier_id"`
	OccurredAt    *string `json:"occurred_at"`
	// Rateio opcional do lancamento entre eventos (substitui o anterior).
	EventAllocations []EventAllocationInput `json:"event_allocations"`
	// Contas a pagar (opcional).
	DueDate    *string `json:"due_date"`
	CostCenter *string `json:"cost_center"`
	Paid       *bool   `json:"paid"`
}

// Update edita um lancamento EM VIGOR, preservando id, entry_seq, recibo e
// anexos. Diferente do fluxo antigo (excluir + recriar), aqui a linha e alterada
// por fin_tx_edit (SECURITY DEFINER), que tambem recalcula a hash-chain. A trava
// de periodo conciliado continua valendo.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, tenantID, id string, in UpdateTxnInput) error {
	if in.CategoryID == nil || *in.CategoryID == "" {
		return ErrCategoryRequired
	}
	if in.Amount <= 0 {
		return ErrAmountNotPositive
	}
	targetAccountID := strOr(in.AccountID)
	if targetAccountID == "" {
		return ErrAccountRequired
	}
	// O RLS esconde contas fora do escopo; id invalido cai em ErrNoRows.
	var catType string
	if err := tx.QueryRow(ctx, `SELECT type FROM financial_categories WHERE id = $1::uuid`, *in.CategoryID).Scan(&catType); err != nil {
		return err
	}
	if catType != in.Type {
		return fmt.Errorf("a conta selecionada e de %s, mas o lancamento e de %s", catType, in.Type)
	}
	// Identificacao livre so vale sem membro/benfeitor (igual ao Create).
	if (in.DonorMemberID != nil && *in.DonorMemberID != "") ||
		(in.BenefactorID != nil && *in.BenefactorID != "") {
		in.DonorName = nil
	}
	when, err := parseOccurredAt(in.OccurredAt)
	if err != nil {
		return err
	}
	if err := assertTxnMutable(ctx, tx, id); err != nil {
		return err
	}
	if err := assertTxnNotTransfer(ctx, tx, id); err != nil {
		return err
	}

	// Confirma existencia/escopo (RLS), a filial e a conta atual antes de editar.
	var branchID, currentAccountID string
	if err := tx.QueryRow(ctx, `
		SELECT branch_id::text, COALESCE(account_id::text,'')
		FROM financial_transactions WHERE id = $1::uuid`, id).Scan(&branchID, &currentAccountID); err != nil {
		return err
	}
	if err := assertDateOpen(ctx, tx, tenantID, branchID, when.UTC().Format(dateLayout)); err != nil {
		return err
	}
	if err := validateTxnAccount(ctx, tx, targetAccountID, currentAccountID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		SELECT fin_tx_edit($1::uuid, $2, $3, $4::numeric, $5, $6, $7, $8, $9, $10, $11, $12::timestamptz, $13, $14, $15, $16)`,
		id, strOr(in.CategoryID), in.Type, in.Amount, strOr(in.PaymentMethod),
		strOr(in.DonorMemberID), strOr(in.BenefactorID), strOr(in.DonorName), strOr(in.SupplierID),
		in.IsAnonymous, strOr(in.Description), when, strOr(in.AccountID), strOr(in.DueDate),
		strOr(in.CostCenter), in.Paid != nil && *in.Paid); err != nil {
		return err
	}

	// Rateio de eventos: substitui o conjunto anterior.
	if _, err := tx.Exec(ctx, `DELETE FROM financial_event_allocations WHERE transaction_id = $1::uuid`, id); err != nil {
		return err
	}
	if len(in.EventAllocations) > 0 {
		if err := r.insertEventAllocations(ctx, tx, tenantID, branchID, id, in.EventAllocations); err != nil {
			return err
		}
	}

	// Split de repasses: apaga os antigos e reaplica (somente entradas).
	if _, err := tx.Exec(ctx, `DELETE FROM transfers WHERE financial_transaction_id = $1::uuid`, id); err != nil {
		return err
	}
	if in.Type == "income" {
		if err := r.ApplySplit(ctx, tx, tenantID, branchID, id, in.Amount); err != nil {
			return err
		}
	}

	// Recibo: o corpo do hash mudou; atualiza o documento emitido (mesmo token).
	t, err := r.txnForReceipt(ctx, tx, id)
	if err != nil {
		return err
	}
	return r.updateReceipt(ctx, tx, tenantID, *t)
}

// txnForReceipt carrega os campos do lancamento usados na montagem do recibo.
func (r *Repo) txnForReceipt(ctx context.Context, tx pgx.Tx, id string) (*Transaction, error) {
	var t Transaction
	err := tx.QueryRow(ctx, `
		SELECT id::text, branch_id::text, type, amount::float8, currency, hash,
		       donor_member_id::text, benefactor_id::text, donor_name, is_anonymous,
		       description, occurred_at
		FROM financial_transactions WHERE id = $1::uuid`, id).
		Scan(&t.ID, &t.BranchID, &t.Type, &t.Amount, &t.Currency, &t.Hash,
			&t.DonorMemberID, &t.BenefactorID, &t.DonorName, &t.IsAnonymous,
			&t.Description, &t.OccurredAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// strOr devolve o valor apontado ou "" (NULL no SQL via NULLIF).
func strOr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// Reorder renumera os lancamentos informados (1..N) na ordem exata do slice,
// dentro do escopo do primeiro (tenant, filial, conta, data). Todos os ids devem
// pertencer ao mesmo escopo (mesma conta e mesmo dia). A sequencia e apenas de
// ordenacao - nao entra na hash-chain -, entao o guard append-only nao bloqueia
// (somente entry_seq muda).
func (r *Repo) Reorder(ctx context.Context, tx pgx.Tx, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text, tenant_id::text, branch_id::text, COALESCE(account_id::text,''),
		       (occurred_at AT TIME ZONE 'UTC')::date::text
		FROM financial_transactions
		WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	var scTenant, scBranch, scAccount, scDate string
	first := true
	count := 0
	for rows.Next() {
		var id, tenant, branch, account, date string
		if err := rows.Scan(&id, &tenant, &branch, &account, &date); err != nil {
			return err
		}
		if first {
			scTenant, scBranch, scAccount, scDate = tenant, branch, account, date
			first = false
		} else if tenant != scTenant || branch != scBranch || account != scAccount || date != scDate {
			return fmt.Errorf("os lancamentos devem ser da mesma conta e da mesma data")
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()
	if count != len(ids) {
		return pgx.ErrNoRows
	}
	for _, id := range ids {
		if err := assertTxnMutable(ctx, tx, id); err != nil {
			return err
		}
	}
	// Mesmo lock usado no Create, para nao correr com insercoes.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		scTenant+"|"+scBranch+"|"+scAccount+"|"+scDate); err != nil {
		return err
	}
	for i, id := range ids {
		if _, err := tx.Exec(ctx,
			`UPDATE financial_transactions SET entry_seq = $2 WHERE id = $1::uuid`, id, i+1); err != nil {
			return err
		}
	}
	return nil
}

// insertEventAllocations grava o rateio do lancamento entre eventos. Valores nao
// informados dividem o restante igualmente; eventos fora do escopo sao recusados.
//
// A aritmetica (soma dos fixos, divisao do restante e arredondamento) roda em
// SQL com `numeric` - dinheiro nunca deve ser dividido em ponto flutuante, senao
// a soma dos rateios nao fecha com o total (FIN-1).
func (r *Repo) insertEventAllocations(ctx context.Context, tx pgx.Tx, tenantID, branchID, txID string, allocs []EventAllocationInput) error {
	if len(allocs) == 0 {
		return nil
	}
	eventIDs := make([]string, len(allocs))
	amounts := make([]string, len(allocs))
	for i, a := range allocs {
		eventIDs[i] = a.EventID
		if a.Amount != nil {
			amounts[i] = strconv.FormatFloat(*a.Amount, 'f', 2, 64)
		}
	}
	rows, err := tx.Query(ctx, `
		WITH inp AS (
			SELECT unnest($4::text[])::uuid AS event_id,
			       NULLIF(unnest($5::text[]), '')::numeric AS amount
		),
		agg AS (
			SELECT COALESCE(SUM(amount), 0) AS fixed,
			       COUNT(*) FILTER (WHERE amount IS NULL) AS missing
			FROM inp
		),
		tot AS (
			SELECT amount FROM financial_transactions WHERE id = $3::uuid
		)
		INSERT INTO financial_event_allocations (tenant_id, branch_id, transaction_id, event_id, amount)
		SELECT $1::uuid, $2::uuid, $3::uuid, e.id,
		       COALESCE(i.amount, GREATEST(ROUND((t.amount - a.fixed) / NULLIF(a.missing, 0), 2), 0))
		FROM inp i
		CROSS JOIN agg a
		CROSS JOIN tot t
		JOIN church_events e ON e.id = i.event_id AND e.tenant_id = $1::uuid AND e.branch_id = $2::uuid
		RETURNING event_id`,
		tenantID, branchID, txID, eventIDs, amounts)
	if err != nil {
		return err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		n++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if n != len(allocs) {
		return errors.New("um ou mais eventos do rateio nao pertencem ao escopo")
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
	if err := assertTxnMutable(ctx, tx, id); err != nil {
		return err
	}
	if err := assertTxnNotTransfer(ctx, tx, id); err != nil {
		return err
	}
	if err := assertTxnNotGrouped(ctx, tx, id); err != nil {
		return err
	}
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
	if err := assertTxnMutable(ctx, tx, id); err != nil {
		return err
	}
	if err := assertTxnNotTransfer(ctx, tx, id); err != nil {
		return err
	}
	if err := assertTxnNotGrouped(ctx, tx, id); err != nil {
		return err
	}
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

// List retorna lancamentos do escopo, com filtro opcional de tipo e de periodo
// (from/to no formato AAAA-MM-DD, inclusivos). Datas vazias nao filtram.
func (r *Repo) List(ctx context.Context, tx pgx.Tx, kind, from, to string) ([]Transaction, error) {
	rows, err := tx.Query(ctx, `
		WITH recent AS (
		    SELECT id
		    FROM financial_transactions
		    WHERE ($1 = '' OR type = $1)
		      -- Intervalo sargavel (usa idx_fin_tx_tenant) mantendo a semantica
		      -- de data em UTC usada na sequencia (entry_seq).
		      AND ($2 = '' OR occurred_at >= ($2::date::timestamp AT TIME ZONE 'UTC'))
		      AND ($3 = '' OR occurred_at < (($3::date + 1)::timestamp AT TIME ZONE 'UTC'))
		    ORDER BY occurred_at DESC, created_at DESC, id DESC
		)
		SELECT t.id::text, t.branch_id::text, t.category_id::text, c.name,
		       t.account_id::text, a.name,
		       t.supplier_id::text, s.name,
		       t.donor_member_id::text, t.benefactor_id::text,
		       COALESCE(t.donor_name, dm.full_name, db.name),
		       t.type, t.amount::float8, t.currency, t.payment_method,
		       t.is_anonymous, t.description, t.receipt_issued, t.hash, t.occurred_at, t.created_at, t.entry_seq,
		       d.id::text, d.document_ref, d.qr_token,
		       t.voided_at, t.void_reason,
		       (SELECT count(*) FROM financial_attachments fa WHERE fa.transaction_id = t.id)::int,
		       (SELECT fa.file_url FROM financial_attachments fa WHERE fa.transaction_id = t.id ORDER BY fa.created_at LIMIT 1),
		       t.due_date::text, t.paid_at, t.cost_center, t.reconciled_at,
		       t.group_id::text, g.status,
		       t.account_transfer_id::text, at.kind,
		       (SELECT fa.name FROM account_transfers tr2
		          JOIN financial_accounts fa
		            ON fa.id = CASE WHEN tr2.from_account_id = t.account_id
		                            THEN tr2.to_account_id ELSE tr2.from_account_id END
		         WHERE tr2.id = t.account_transfer_id)
		FROM financial_transactions t
		JOIN recent r ON r.id = t.id
		LEFT JOIN financial_categories c ON c.id = t.category_id
		LEFT JOIN financial_accounts a ON a.id = t.account_id
		LEFT JOIN suppliers s ON s.id = t.supplier_id
		LEFT JOIN members dm ON dm.id = t.donor_member_id
		LEFT JOIN benefactors db ON db.id = t.benefactor_id
		LEFT JOIN documents d ON d.kind = 'receipt' AND d.content->>'tx_id' = t.id::text
		LEFT JOIN financial_transaction_groups g ON g.id = t.group_id
		LEFT JOIN account_transfers at ON at.id = t.account_transfer_id
		ORDER BY t.occurred_at, t.entry_seq NULLS LAST, t.created_at, t.id`, kind, from, to)
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
			&t.DonorMemberID, &t.BenefactorID, &t.DonorName,
			&t.Type, &t.Amount, &t.Currency, &t.PaymentMethod,
			&t.IsAnonymous, &t.Description, &t.ReceiptIssued, &t.Hash, &t.OccurredAt, &t.CreatedAt, &t.EntrySeq,
			&t.ReceiptID, &t.ReceiptRef, &t.ReceiptToken,
			&t.VoidedAt, &t.VoidReason, &t.AttachmentCount, &t.AttachmentURL,
			&t.DueDate, &t.PaidAt, &t.CostCenter, &t.ReconciledAt,
			&t.GroupID, &t.GroupStatus,
			&t.AccountTransferID, &t.TransferKind, &t.CounterpartAccountName); err != nil {
			return nil, err
		}
		t.Reconciled = t.ReconciledAt != nil
		out = append(out, t)
	}
	return out, rows.Err()
}

// Settle marca (paid=true) ou desmarca (paid=false) a quitacao do lancamento.
// Usado pelas contas a pagar. Colunas fora da hash-chain, entao o trigger
// append-only permite o UPDATE.
func (r *Repo) Settle(ctx context.Context, tx pgx.Tx, id string, paid bool) error {
	if err := assertTxnMutable(ctx, tx, id); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE financial_transactions
		SET paid_at = CASE WHEN $2 THEN now() ELSE NULL END
		WHERE id = $1::uuid`, id, paid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ReconcileTransactions concilia/desconcilia um conjunto de lancamentos na mesma
// transacao (atomico). Usado pelo atalho "conciliar pendentes" do periodo.
func (r *Repo) ReconcileTransactions(ctx context.Context, tx pgx.Tx, ids []string, reconciled bool, actorID string) error {
	for _, id := range ids {
		if err := r.ReconcileTransaction(ctx, tx, id, reconciled, actorID); err != nil {
			return err
		}
	}
	return nil
}

// ReconcileTransaction concilia/desconcilia um unico lancamento. A RPC no banco
// valida o tenant e as travas de periodo/auditoria.
func (r *Repo) ReconcileTransaction(ctx context.Context, tx pgx.Tx, id string, reconciled bool, actorID string) error {
	var tenantID, branchID, date string
	var reconciledAt, voidedAt *time.Time
	var groupID *string
	err := tx.QueryRow(ctx, `
		SELECT tenant_id::text, branch_id::text,
		       (occurred_at AT TIME ZONE 'UTC')::date::text,
		       reconciled_at, voided_at, group_id::text
		FROM financial_transactions WHERE id = $1::uuid`, id).
		Scan(&tenantID, &branchID, &date, &reconciledAt, &voidedAt, &groupID)
	if err != nil {
		return err
	}
	if voidedAt != nil {
		return ErrTransactionVoided
	}
	if groupID != nil {
		return ErrGroupedReconcile
	}
	if (reconciledAt != nil) == reconciled {
		return nil
	}
	if err := assertDateOpen(ctx, tx, tenantID, branchID, date); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SELECT fin_tx_reconcile($1::uuid, $2, $3)`, id, reconciled, actorID)
	return err
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
			  AND account_transfer_id IS NULL
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
	// Nunca devolver nil: `by_category` e serializado como null e quebra os
	// consumidores que chamam .filter/.map direto no JSON.
	b.ByCategory = []BalanceItem{}

	rows, err := tx.Query(ctx, `
		SELECT COALESCE(t.category_id::text,''), COALESCE(c.code,''),
		       COALESCE(c.name,'Sem categoria'), t.type, SUM(t.amount)::float8
		FROM financial_transactions t
		LEFT JOIN financial_categories c ON c.id = t.category_id
		WHERE ($1 = '' OR t.type = $1)
		  AND t.voided_at IS NULL
		  AND t.account_transfer_id IS NULL
		GROUP BY 1,2,3,4
		ORDER BY t.type,
		         NULLIF(regexp_replace(COALESCE(c.code,''), '\D', '', 'g'), '')::int NULLS LAST,
		         COALESCE(c.code,'')`, kind)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	for rows.Next() {
		var it BalanceItem
		if err := rows.Scan(&it.CategoryID, &it.Code, &it.Category, &it.Type, &it.Total); err != nil {
			return b, err
		}
		b.ByCategory = append(b.ByCategory, it)
	}
	if err := rows.Err(); err != nil {
		return b, err
	}
	rows.Close()

	// Saldo por tipo de conta (posicao de caixa). Inclui as transferencias entre
	// contas, que alteram o saldo de cada conta mas somam zero no total.
	b.ByAccountType = []AccountTypeBalance{}
	acctRows, err := tx.Query(ctx, `
		WITH acct AS (
			SELECT id, account_type, initial_balance
			FROM financial_accounts
			WHERE is_active
		),
		agg AS (
			SELECT account_id,
			       COALESCE(SUM(amount) FILTER (WHERE type='income'),0) AS inc,
			       COALESCE(SUM(amount) FILTER (WHERE type='expense'),0) AS exp
			FROM financial_transactions
			WHERE voided_at IS NULL
			GROUP BY account_id
		)
		SELECT acct.account_type,
		       COALESCE(SUM(acct.initial_balance),0)::float8,
		       COALESCE(SUM(agg.inc),0)::float8,
		       COALESCE(SUM(agg.exp),0)::float8
		FROM acct
		LEFT JOIN agg ON agg.account_id = acct.id
		GROUP BY acct.account_type
		ORDER BY acct.account_type`)
	if err != nil {
		return b, err
	}
	defer acctRows.Close()
	for acctRows.Next() {
		var a AccountTypeBalance
		if err := acctRows.Scan(&a.AccountType, &a.InitialBalance, &a.Income, &a.Expense); err != nil {
			return b, err
		}
		a.Net = a.InitialBalance + a.Income - a.Expense
		b.ByAccountType = append(b.ByAccountType, a)
	}
	return b, acctRows.Err()
}

func nullIfEmpty(s *string) *string {
	if s != nil && *s == "" {
		return nil
	}
	return s
}
