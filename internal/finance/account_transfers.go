package finance

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Erros de transferencia entre contas.
var (
	ErrAccountTransferNotFound   = errors.New("transferencia entre contas nao encontrada")
	ErrAccountTransferSame       = errors.New("as contas de origem e destino precisam ser diferentes")
	ErrAccountTransferKind       = errors.New("tipo de transferencia incompativel com as contas")
	ErrAccountTransferReconciled = errors.New("transferencia com perna conciliada: desconcilie antes de excluir")
	ErrAccountTransferGrouped    = errors.New("transferencia com perna agrupada: desagrupe antes de excluir")
	ErrTransactionIsTransfer     = errors.New("lancamento e perna de transferencia entre contas; use a tela de transferencias")
)

// AccountTransfer e uma movimentacao entre contas (deposito/resgate). As duas
// pernas em financial_transactions compartilham o mesmo id.
type AccountTransfer struct {
	ID                string  `json:"id"`
	BranchID          string  `json:"branch_id"`
	FromAccountID     string  `json:"from_account_id"`
	FromAccountName   *string `json:"from_account_name,omitempty"`
	ToAccountID       string  `json:"to_account_id"`
	ToAccountName     *string `json:"to_account_name,omitempty"`
	Kind              string  `json:"kind"`
	Amount            float64 `json:"amount"`
	OccurredAt        string  `json:"occurred_at"`
	Description       *string `json:"description,omitempty"`
	CreatedAt         string  `json:"created_at"`
	FromTransactionID *string `json:"from_transaction_id,omitempty"`
	ToTransactionID   *string `json:"to_transaction_id,omitempty"`
	FromReconciled    bool    `json:"from_reconciled"`
	ToReconciled      bool    `json:"to_reconciled"`
}

type CreateAccountTransferInput struct {
	FromAccountID string  `json:"from_account_id"`
	ToAccountID   string  `json:"to_account_id"`
	Kind          string  `json:"kind"`
	Amount        float64 `json:"amount"`
	OccurredAt    string  `json:"occurred_at"`
	Description   string  `json:"description"`
}

func normalizeTransferKind(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", "deposit", "deposito", "aplicacao", "aplicar":
		return "deposit", nil
	case "redeem", "resgate", "resgatar":
		return "redeem", nil
	case "transfer", "transferencia":
		return "transfer", nil
	default:
		return "", errors.New("tipo de transferencia invalido (use deposit, redeem ou transfer)")
	}
}

type transferAccount struct {
	ID          string
	Name        string
	AccountType string
	BranchID    string
	IsActive    bool
}

func (r *Repo) loadTransferAccount(ctx context.Context, tx pgx.Tx, id string) (*transferAccount, error) {
	var a transferAccount
	if err := tx.QueryRow(ctx, `
		SELECT id::text, name, account_type, COALESCE(branch_id::text,''), is_active
		FROM financial_accounts WHERE id = $1::uuid`, id).
		Scan(&a.ID, &a.Name, &a.AccountType, &a.BranchID, &a.IsActive); err != nil {
		return nil, err
	}
	return &a, nil
}

// CreateAccountTransfer cria a transferencia e as duas pernas (saida na origem,
// entrada no destino). As pernas nao geram recibo nem split de repasses.
func (r *Repo) CreateAccountTransfer(ctx context.Context, tx pgx.Tx, tenantID, actorID, defaultBranchID string, in CreateAccountTransferInput) (*AccountTransfer, error) {
	fromID := strings.TrimSpace(in.FromAccountID)
	toID := strings.TrimSpace(in.ToAccountID)
	if fromID == "" || toID == "" {
		return nil, errors.New("selecione as contas de origem e destino")
	}
	if fromID == toID {
		return nil, ErrAccountTransferSame
	}
	if in.Amount <= 0 {
		return nil, ErrAmountNotPositive
	}
	kind, err := normalizeTransferKind(in.Kind)
	if err != nil {
		return nil, err
	}
	from, err := r.loadTransferAccount(ctx, tx, fromID)
	if err != nil {
		return nil, err
	}
	to, err := r.loadTransferAccount(ctx, tx, toID)
	if err != nil {
		return nil, err
	}
	if !from.IsActive || !to.IsActive {
		return nil, errors.New("as contas da transferencia precisam estar ativas")
	}
	// Contas globais (branch_id NULL, criadas pela Sede) sao compativeis com
	// qualquer filial. Se as duas tiverem filial, precisam ser a mesma.
	if from.BranchID != "" && to.BranchID != "" && from.BranchID != to.BranchID {
		return nil, errors.New("as contas precisam ser da mesma filial")
	}
	branchID := from.BranchID
	if branchID == "" {
		branchID = to.BranchID
	}
	if branchID == "" {
		branchID = strings.TrimSpace(defaultBranchID)
	}
	if branchID == "" {
		// Fallback: usa a Matriz (ou a primeira filial) do tenant.
		if err := tx.QueryRow(ctx, `
			SELECT id::text FROM branches WHERE tenant_id = $1::uuid
			ORDER BY (kind = 'matriz') DESC, name LIMIT 1`, tenantID).Scan(&branchID); err != nil {
			return nil, errors.New("selecione a filial da transferencia")
		}
	}

	switch kind {
	case "deposit":
		if from.AccountType == "investment" || to.AccountType != "investment" {
			return nil, ErrAccountTransferKind
		}
	case "redeem":
		if from.AccountType != "investment" || to.AccountType == "investment" {
			return nil, ErrAccountTransferKind
		}
	}

	when, err := parseOccurredAt(&in.OccurredAt)
	if err != nil {
		return nil, err
	}
	date := when.UTC().Format(dateLayout)
	if err := assertDateOpen(ctx, tx, tenantID, branchID, date); err != nil {
		return nil, err
	}

	// Serializa criacoes de transferencia no tenant (evita deadlock entre
	// transferencias inversas simultaneas).
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		tenantID+"|account-transfer"); err != nil {
		return nil, err
	}

	desc := strings.TrimSpace(in.Description)
	if desc == "" {
		switch kind {
		case "deposit":
			desc = "Deposito em " + to.Name
		case "redeem":
			desc = "Resgate de " + from.Name
		default:
			desc = "Transferencia de " + from.Name + " para " + to.Name
		}
	}

	var transferID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO account_transfers
			(tenant_id, branch_id, from_account_id, to_account_id, kind, amount, occurred_at, description, created_by)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7::date, NULLIF($8,''), NULLIF($9,'')::uuid)
		RETURNING id::text`,
		tenantID, branchID, fromID, toID, kind, in.Amount, date, desc, actorID).Scan(&transferID); err != nil {
		return nil, err
	}

	// Perna de saida (origem) e de entrada (destino).
	if _, _, _, _, err := r.Create(ctx, tx, tenantID, branchID, CreateTxnInput{
		Type:              "expense",
		Amount:            in.Amount,
		AccountID:         &fromID,
		Description:       &desc,
		OccurredAt:        &date,
		ActorID:           actorID,
		AccountTransferID: &transferID,
	}); err != nil {
		return nil, err
	}
	if _, _, _, _, err := r.Create(ctx, tx, tenantID, branchID, CreateTxnInput{
		Type:              "income",
		Amount:            in.Amount,
		AccountID:         &toID,
		Description:       &desc,
		OccurredAt:        &date,
		ActorID:           actorID,
		AccountTransferID: &transferID,
	}); err != nil {
		return nil, err
	}

	return r.GetAccountTransfer(ctx, tx, transferID)
}

const accountTransferCols = `t.id::text, t.branch_id::text,
	t.from_account_id::text, fa.name, t.to_account_id::text, ta.name,
	t.kind, t.amount::float8, to_char(t.occurred_at,'YYYY-MM-DD'), t.description,
	to_char(t.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
	fe.id::text, ti.id::text,
	COALESCE(fe.reconciled_at IS NOT NULL, false),
	COALESCE(ti.reconciled_at IS NOT NULL, false)`

const accountTransferJoin = `
	FROM account_transfers t
	JOIN financial_accounts fa ON fa.id = t.from_account_id
	JOIN financial_accounts ta ON ta.id = t.to_account_id
	LEFT JOIN LATERAL (
	    SELECT id, reconciled_at FROM financial_transactions
	    WHERE account_transfer_id = t.id AND type = 'expense'
	    ORDER BY created_at, id LIMIT 1
	) fe ON true
	LEFT JOIN LATERAL (
	    SELECT id, reconciled_at FROM financial_transactions
	    WHERE account_transfer_id = t.id AND type = 'income'
	    ORDER BY created_at, id LIMIT 1
	) ti ON true`

func scanAccountTransfer(row pgx.Row) (*AccountTransfer, error) {
	var t AccountTransfer
	if err := row.Scan(&t.ID, &t.BranchID,
		&t.FromAccountID, &t.FromAccountName, &t.ToAccountID, &t.ToAccountName,
		&t.Kind, &t.Amount, &t.OccurredAt, &t.Description, &t.CreatedAt,
		&t.FromTransactionID, &t.ToTransactionID, &t.FromReconciled, &t.ToReconciled); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repo) GetAccountTransfer(ctx context.Context, tx pgx.Tx, id string) (*AccountTransfer, error) {
	t, err := scanAccountTransfer(tx.QueryRow(ctx, `SELECT `+accountTransferCols+accountTransferJoin+` WHERE t.id = $1::uuid`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountTransferNotFound
		}
		return nil, err
	}
	return t, nil
}

// ListAccountTransfers lista as transferencias do escopo com filtros opcionais.
func (r *Repo) ListAccountTransfers(ctx context.Context, tx pgx.Tx, from, to, accountID string) ([]AccountTransfer, error) {
	rows, err := tx.Query(ctx, `SELECT `+accountTransferCols+accountTransferJoin+`
		WHERE ($1 = '' OR t.occurred_at >= $1::date)
		  AND ($2 = '' OR t.occurred_at <= $2::date)
		  AND ($3 = '' OR t.from_account_id = $3::uuid OR t.to_account_id = $3::uuid)
		ORDER BY t.occurred_at DESC, t.created_at DESC, t.id DESC`, from, to, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AccountTransfer{}
	for rows.Next() {
		var t AccountTransfer
		if err := rows.Scan(&t.ID, &t.BranchID,
			&t.FromAccountID, &t.FromAccountName, &t.ToAccountID, &t.ToAccountName,
			&t.Kind, &t.Amount, &t.OccurredAt, &t.Description, &t.CreatedAt,
			&t.FromTransactionID, &t.ToTransactionID, &t.FromReconciled, &t.ToReconciled); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DeleteAccountTransfer remove as duas pernas e a transferencia. Bloqueia se
// alguma perna estiver conciliada/agrupada ou em periodo encerrado. Entradas do
// extrato bancario vinculadas voltam para "pendente" (sem correspondencia).
func (r *Repo) DeleteAccountTransfer(ctx context.Context, tx pgx.Tx, tenantID, id string) error {
	var branchID, date string
	if err := tx.QueryRow(ctx, `
		SELECT branch_id::text, occurred_at::text FROM account_transfers WHERE id = $1::uuid`, id).
		Scan(&branchID, &date); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountTransferNotFound
		}
		return err
	}

	rows, err := tx.Query(ctx, `
		SELECT id::text, reconciled_at, group_id::text
		FROM financial_transactions WHERE account_transfer_id = $1::uuid`, id)
	if err != nil {
		return err
	}
	ids := []string{}
	reconciled := false
	grouped := false
	for rows.Next() {
		var txID string
		var reconciledAt *time.Time
		var groupID *string
		if err := rows.Scan(&txID, &reconciledAt, &groupID); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, txID)
		if reconciledAt != nil {
			reconciled = true
		}
		if groupID != nil {
			grouped = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ids) == 0 {
		return ErrAccountTransferNotFound
	}
	if reconciled {
		return ErrAccountTransferReconciled
	}
	if grouped {
		return ErrAccountTransferGrouped
	}
	if err := assertDateOpen(ctx, tx, tenantID, branchID, date); err != nil {
		return err
	}

	// Desvincula o extrato bancario (volta a pendente para ser conciliado de novo).
	importRows, err := tx.Query(ctx, `
		SELECT DISTINCT import_id::text FROM bank_statement_entries
		WHERE transaction_id = ANY($1::uuid[])`, ids)
	if err != nil {
		return err
	}
	importIDs := []string{}
	for importRows.Next() {
		var impID string
		if err := importRows.Scan(&impID); err != nil {
			importRows.Close()
			return err
		}
		importIDs = append(importIDs, impID)
	}
	importRows.Close()
	if err := importRows.Err(); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE bank_statement_entries
		SET status = 'pendente', transaction_id = NULL, group_id = NULL, updated_at = now()
		WHERE transaction_id = ANY($1::uuid[])`, ids); err != nil {
		return err
	}
	for _, impID := range importIDs {
		if err := refreshBankCounters(ctx, tx, impID); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM financial_transactions WHERE id = ANY($1::uuid[])`, ids); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM account_transfers WHERE id = $1::uuid`, id); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SELECT fin_tx_rechain($1::uuid)`, tenantID)
	return err
}

// assertTxnNotTransfer impede editar/excluir/estornar uma perna isoladamente.
func assertTxnNotTransfer(ctx context.Context, tx pgx.Tx, id string) error {
	var transferID *string
	if err := tx.QueryRow(ctx, `
		SELECT account_transfer_id::text FROM financial_transactions WHERE id = $1::uuid`, id).
		Scan(&transferID); err != nil {
		return err
	}
	if transferID != nil {
		return ErrTransactionIsTransfer
	}
	return nil
}
