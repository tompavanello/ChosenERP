package finance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Erros de agrupamento de lancamentos.
var (
	ErrGroupNotFound      = errors.New("grupo de lancamentos nao encontrado")
	ErrGroupMinMembers    = errors.New("grupo precisa de pelo menos 2 lancamentos")
	ErrGroupMixed         = errors.New("todos os lancamentos precisam ter a mesma filial, conta bancaria e tipo")
	ErrGroupReconciled    = errors.New("grupo conciliado: nao pode ser alterado ou desagrupado")
	ErrGroupDateMismatch  = errors.New("todos os lancamentos e o grupo precisam ter a mesma data")
	ErrTransactionGrouped = errors.New("lancamento ja pertence a um grupo")
)

// TransactionGroup e uma movimentacao bancaria que reune varios lancamentos.
// O total e o numero de membros sao calculados a partir dos lancamentos.
type TransactionGroup struct {
	ID           string        `json:"id"`
	BranchID     string        `json:"branch_id"`
	AccountID    string        `json:"account_id"`
	AccountName  *string       `json:"account_name,omitempty"`
	Type         string        `json:"type"`
	OccurredAt   string        `json:"occurred_at"`
	Description  *string       `json:"description,omitempty"`
	Status       string        `json:"status"`
	Reconciled   bool          `json:"reconciled"`
	ReconciledAt *time.Time    `json:"reconciled_at,omitempty"`
	BankEntryID  *string       `json:"bank_entry_id,omitempty"`
	EntrySeq     *int          `json:"entry_seq,omitempty"`
	TotalAmount  float64       `json:"total_amount"`
	MemberCount  int           `json:"member_count"`
	CreatedAt    time.Time     `json:"created_at"`
	Members      []GroupMember `json:"members,omitempty"`
}

// GroupMember e o detalhe contabil de um lancamento que compoe o grupo.
type GroupMember struct {
	ID           string     `json:"id"`
	CategoryID   *string    `json:"category_id,omitempty"`
	CategoryName *string    `json:"category_name,omitempty"`
	Type         string     `json:"type"`
	Amount       float64    `json:"amount"`
	Description  *string    `json:"description,omitempty"`
	DonorName    *string    `json:"donor_name,omitempty"`
	OccurredAt   string     `json:"occurred_at"`
	Reconciled   bool       `json:"reconciled"`
	ReconciledAt *time.Time `json:"reconciled_at,omitempty"`
	EntrySeq     *int       `json:"entry_seq,omitempty"`
	ReceiptID    *string    `json:"receipt_id,omitempty"`
	VoidedAt     *time.Time `json:"voided_at,omitempty"`
}

type CreateGroupInput struct {
	TransactionIDs []string `json:"transaction_ids"`
	OccurredAt     string   `json:"occurred_at"`
	Description    *string  `json:"description"`
	AccountID      *string  `json:"account_id"`
}

type UpdateGroupInput struct {
	OccurredAt  *string `json:"occurred_at,omitempty"`
	Description *string `json:"description,omitempty"`
}

func uniqueIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// CreateGroup cria o grupo e anexa os lancamentos informados na mesma transacao.
// A conta/filial/tipo/data sao derivados dos lancamentos (precisam ser iguais).
func (r *Repo) CreateGroup(ctx context.Context, tx pgx.Tx, tenantID, actorID string, in CreateGroupInput) (*TransactionGroup, error) {
	ids := uniqueIDs(in.TransactionIDs)
	if len(ids) < 2 {
		return nil, ErrGroupMinMembers
	}
	requestedDate := strings.TrimSpace(in.OccurredAt)
	if requestedDate != "" {
		if _, err := time.ParseInLocation(dateLayout, requestedDate, time.UTC); err != nil {
			return nil, errors.New("data do grupo invalida")
		}
	}

	rows, err := tx.Query(ctx, `
		SELECT id::text, tenant_id::text, branch_id::text, COALESCE(account_id::text,''),
		       type, amount::float8, voided_at, reconciled_at, group_id::text,
		       (occurred_at AT TIME ZONE 'UTC')::date::text
		FROM financial_transactions
		WHERE id = ANY($1::uuid[])
		FOR UPDATE`, ids)
	if err != nil {
		return nil, err
	}

	var rowTenant, branch, account, kind, memberDate string
	found := 0
	for rows.Next() {
		var id, tTenant, tBranch, tAccount, tType, tDate string
		var amount float64
		var voidedAt, reconciledAt *time.Time
		var groupID *string
		if err := rows.Scan(&id, &tTenant, &tBranch, &tAccount, &tType, &amount,
			&voidedAt, &reconciledAt, &groupID, &tDate); err != nil {
			rows.Close()
			return nil, err
		}
		if found == 0 {
			rowTenant, branch, account, kind, memberDate = tTenant, tBranch, tAccount, tType, tDate
		}
		if tTenant != rowTenant || tBranch != branch || tAccount != account || tType != kind {
			rows.Close()
			return nil, ErrGroupMixed
		}
		if tDate != memberDate {
			rows.Close()
			return nil, fmt.Errorf("%w: ha lancamentos em %s e em %s", ErrGroupDateMismatch, memberDate, tDate)
		}
		if voidedAt != nil {
			rows.Close()
			return nil, errors.New("lancamento estornado nao pode ser agrupado")
		}
		if reconciledAt != nil {
			rows.Close()
			return nil, ErrTransactionReconciled
		}
		if groupID != nil {
			rows.Close()
			return nil, ErrTransactionGrouped
		}
		if account == "" {
			rows.Close()
			return nil, ErrAccountRequired
		}
		found++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if found != len(ids) {
		return nil, errors.New("um ou mais lancamentos nao foram encontrados")
	}
	if rowTenant != tenantID {
		return nil, errors.New("lancamento fora do tenant")
	}
	if in.AccountID != nil && strings.TrimSpace(*in.AccountID) != "" &&
		strings.TrimSpace(*in.AccountID) != account {
		return nil, ErrGroupMixed
	}
	if requestedDate != "" && requestedDate != memberDate {
		return nil, fmt.Errorf("%w: o grupo esta em %s, mas os lancamentos estao em %s",
			ErrGroupDateMismatch, requestedDate, memberDate)
	}
	if err := assertDateOpen(ctx, tx, tenantID, branch, memberDate); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		tenantID+"|"+branch+"|"+account+"|"+memberDate); err != nil {
		return nil, err
	}
	entrySeq, err := allocateGroupEntrySeq(ctx, tx, ids, tenantID, branch, account, memberDate)
	if err != nil {
		return nil, err
	}

	var groupID string
	err = tx.QueryRow(ctx, `
		INSERT INTO financial_transaction_groups
			(tenant_id, branch_id, account_id, type, occurred_at, description, created_by, entry_seq)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5::date, NULLIF($6,''), NULLIF($7,'')::uuid, $8)
		RETURNING id::text`,
		tenantID, branch, account, kind, memberDate, in.Description, actorID, entrySeq).Scan(&groupID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `SELECT fin_group_attach($1::uuid, $2::uuid[])`, groupID, ids); err != nil {
		return nil, err
	}
	return r.GetGroup(ctx, tx, groupID)
}

// ListGroups lista os grupos do escopo, com filtro opcional de periodo/status.
func (r *Repo) ListGroups(ctx context.Context, tx pgx.Tx, from, to, status string) ([]TransactionGroup, error) {
	rows, err := tx.Query(ctx, `
		SELECT g.id::text, g.branch_id::text, g.account_id::text, a.name,
		       g.type, to_char(g.occurred_at,'YYYY-MM-DD'), g.description,
		       g.status, g.reconciled_at, g.bank_entry_id::text, g.created_at, g.entry_seq,
		       COALESCE(agg.total,0)::float8, COALESCE(agg.n,0)::int
		FROM financial_transaction_groups g
		LEFT JOIN financial_accounts a ON a.id = g.account_id
		LEFT JOIN LATERAL (
		    SELECT SUM(t.amount) AS total, count(*) AS n
		    FROM financial_transactions t
		    WHERE t.group_id = g.id AND t.voided_at IS NULL
		) agg ON true
		WHERE ($1 = '' OR g.occurred_at >= $1::date)
		  AND ($2 = '' OR g.occurred_at <= $2::date)
		  AND ($3 = '' OR g.status = $3)
		ORDER BY g.occurred_at DESC, g.entry_seq NULLS LAST, g.created_at DESC`, from, to, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TransactionGroup{}
	for rows.Next() {
		var g TransactionGroup
		if err := rows.Scan(&g.ID, &g.BranchID, &g.AccountID, &g.AccountName,
			&g.Type, &g.OccurredAt, &g.Description, &g.Status, &g.ReconciledAt,
			&g.BankEntryID, &g.CreatedAt, &g.EntrySeq, &g.TotalAmount, &g.MemberCount); err != nil {
			return nil, err
		}
		g.Reconciled = g.Status == "conciliado"
		out = append(out, g)
	}
	return out, rows.Err()
}

// GetGroup devolve o grupo com os detalhes contabeis de cada membro.
func (r *Repo) GetGroup(ctx context.Context, tx pgx.Tx, id string) (*TransactionGroup, error) {
	var g TransactionGroup
	err := tx.QueryRow(ctx, `
		SELECT g.id::text, g.branch_id::text, g.account_id::text, a.name,
		       g.type, to_char(g.occurred_at,'YYYY-MM-DD'), g.description,
		       g.status, g.reconciled_at, g.bank_entry_id::text, g.created_at, g.entry_seq,
		       COALESCE(agg.total,0)::float8, COALESCE(agg.n,0)::int
		FROM financial_transaction_groups g
		LEFT JOIN financial_accounts a ON a.id = g.account_id
		LEFT JOIN LATERAL (
		    SELECT SUM(t.amount) AS total, count(*) AS n
		    FROM financial_transactions t
		    WHERE t.group_id = g.id AND t.voided_at IS NULL
		) agg ON true
		WHERE g.id = $1::uuid`, id).
		Scan(&g.ID, &g.BranchID, &g.AccountID, &g.AccountName, &g.Type,
			&g.OccurredAt, &g.Description, &g.Status, &g.ReconciledAt,
			&g.BankEntryID, &g.CreatedAt, &g.EntrySeq, &g.TotalAmount, &g.MemberCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	g.Reconciled = g.Status == "conciliado"

	rows, err := tx.Query(ctx, `
		SELECT t.id::text, t.category_id::text, c.name, t.type, t.amount::float8,
		       t.description,
		       COALESCE(t.donor_name, dm.full_name, bf.name),
		       to_char(t.occurred_at,'YYYY-MM-DD'),
		       t.reconciled_at, t.entry_seq,
		       (SELECT d.id::text FROM documents d
		         WHERE d.kind = 'receipt' AND d.content->>'tx_id' = t.id::text
		         ORDER BY d.created_at DESC LIMIT 1),
		       t.voided_at
		FROM financial_transactions t
		LEFT JOIN financial_categories c ON c.id = t.category_id
		LEFT JOIN members dm ON dm.id = t.donor_member_id
		LEFT JOIN benefactors bf ON bf.id = t.benefactor_id
		WHERE t.group_id = $1::uuid
		ORDER BY t.occurred_at, t.entry_seq NULLS LAST, t.created_at, t.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	g.Members = []GroupMember{}
	for rows.Next() {
		var m GroupMember
		if err := rows.Scan(&m.ID, &m.CategoryID, &m.CategoryName, &m.Type,
			&m.Amount, &m.Description, &m.DonorName, &m.OccurredAt,
			&m.ReconciledAt, &m.EntrySeq, &m.ReceiptID, &m.VoidedAt); err != nil {
			return nil, err
		}
		m.Reconciled = m.ReconciledAt != nil
		g.Members = append(g.Members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &g, nil
}

// UpdateGroup altera a data/descricao do grupo (somente enquanto agrupado).
// Mudar a data move todos os membros para a nova data (o grupo e os membros
// precisam compartilhar a mesma data) e recalcula a hash-chain.
func (r *Repo) UpdateGroup(ctx context.Context, tx pgx.Tx, id string, in UpdateGroupInput) (*TransactionGroup, error) {
	var date *string
	if in.OccurredAt != nil {
		d := strings.TrimSpace(*in.OccurredAt)
		if d != "" {
			if _, err := time.ParseInLocation(dateLayout, d, time.UTC); err != nil {
				return nil, errors.New("data do grupo invalida")
			}
			date = &d
		}
	}
	var description *string
	if in.Description != nil {
		d := strings.TrimSpace(*in.Description)
		description = &d
	}

	var curTenant, curBranch, curAccount, curStatus, curDate string
	var bankEntryID *string
	if err := tx.QueryRow(ctx, `
		SELECT tenant_id::text, branch_id::text, account_id::text, status,
		       occurred_at::text, bank_entry_id::text
		FROM financial_transaction_groups WHERE id = $1::uuid`, id).
		Scan(&curTenant, &curBranch, &curAccount, &curStatus, &curDate, &bankEntryID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	if curStatus != "agrupado" {
		return nil, ErrGroupReconciled
	}

	newDate := curDate
	if date != nil && *date != "" {
		newDate = *date
	}
	if newDate != curDate {
		if bankEntryID != nil {
			return nil, errors.New("grupo vinculado ao extrato: a data do deposito nao pode ser alterada")
		}
		if err := assertDateOpen(ctx, tx, curTenant, curBranch, curDate); err != nil {
			return nil, err
		}
		if err := assertDateOpen(ctx, tx, curTenant, curBranch, newDate); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
			curTenant+"|"+curBranch+"|"+curAccount+"|"+newDate); err != nil {
			return nil, err
		}
		newSeq, err := nextLineSeq(ctx, tx, curTenant, curBranch, curAccount, newDate)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx,
			`SELECT fin_group_move_date($1::uuid, $2::date, $3)`, id, newDate, newSeq); err != nil {
			return nil, err
		}
		if description != nil {
			if _, err := tx.Exec(ctx, `
				UPDATE financial_transaction_groups
				SET description = CASE WHEN $2 = '' THEN NULL ELSE $2 END
				WHERE id = $1::uuid`, id, *description); err != nil {
				return nil, err
			}
		}
		return r.GetGroup(ctx, tx, id)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE financial_transaction_groups
		SET description = CASE WHEN $2::text IS NULL THEN description
		                       WHEN $2 = '' THEN NULL
		                       ELSE $2 END
		WHERE id = $1::uuid`, id, description); err != nil {
		return nil, err
	}
	return r.GetGroup(ctx, tx, id)
}

// DeleteGroup desagrupa todos os membros e exclui o grupo. Um grupo conciliado
// precisa ser desconcliado antes.
func (r *Repo) DeleteGroup(ctx context.Context, tx pgx.Tx, id string) error {
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM financial_transaction_groups WHERE id = $1::uuid`, id).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGroupNotFound
		}
		return err
	}
	if status == "conciliado" {
		return ErrGroupReconciled
	}
	rows, err := tx.Query(ctx, `SELECT id::text FROM financial_transactions WHERE group_id = $1::uuid`, id)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var txID string
		if err := rows.Scan(&txID); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, txID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ids) > 0 {
		if _, err := tx.Exec(ctx, `SELECT fin_group_detach($1::uuid, $2::uuid[])`, id, ids); err != nil {
			return err
		}
	}
	tag, err := tx.Exec(ctx, `DELETE FROM financial_transaction_groups WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	for _, txID := range ids {
		if err := assignFreedLineSeq(ctx, tx, txID); err != nil {
			return err
		}
	}
	return nil
}

// AddGroupMembers anexa lancamentos ao grupo.
func (r *Repo) AddGroupMembers(ctx context.Context, tx pgx.Tx, id string, txIDs []string) (*TransactionGroup, error) {
	ids := uniqueIDs(txIDs)
	if len(ids) == 0 {
		return nil, errors.New("informe ao menos um lancamento")
	}
	var groupDate string
	if err := tx.QueryRow(ctx,
		`SELECT occurred_at::text FROM financial_transaction_groups WHERE id = $1::uuid`, id).
		Scan(&groupDate); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	var mismatch int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)::int FROM financial_transactions
		WHERE id = ANY($1::uuid[])
		  AND (occurred_at AT TIME ZONE 'UTC')::date <> $2::date`, ids, groupDate).Scan(&mismatch); err != nil {
		return nil, err
	}
	if mismatch > 0 {
		return nil, fmt.Errorf("%w: o grupo esta em %s", ErrGroupDateMismatch, groupDate)
	}
	if _, err := tx.Exec(ctx, `SELECT fin_group_attach($1::uuid, $2::uuid[])`, id, ids); err != nil {
		return nil, err
	}
	return r.GetGroup(ctx, tx, id)
}

// RemoveGroupMember desanexa um lancamento do grupo. O grupo continua exigindo
// pelo menos 2 membros; para remove-lo por inteiro use DeleteGroup.
func (r *Repo) RemoveGroupMember(ctx context.Context, tx pgx.Tx, id, txID string) (*TransactionGroup, error) {
	if _, err := tx.Exec(ctx, `SELECT fin_group_detach($1::uuid, $2::uuid[])`, id, []string{txID}); err != nil {
		return nil, err
	}
	var remaining int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)::int FROM financial_transactions
		WHERE group_id = $1::uuid AND voided_at IS NULL`, id).Scan(&remaining); err != nil {
		return nil, err
	}
	if remaining < 2 {
		return nil, ErrGroupMinMembers
	}
	if err := assignFreedLineSeq(ctx, tx, txID); err != nil {
		return nil, err
	}
	return r.GetGroup(ctx, tx, id)
}

// ReconcileGroup concilia/desconcilia o grupo e todos os seus membros.
func (r *Repo) ReconcileGroup(ctx context.Context, tx pgx.Tx, id string, reconciled bool, actorID string) (*TransactionGroup, error) {
	if _, err := tx.Exec(ctx, `SELECT fin_group_reconcile($1::uuid, $2, $3)`, id, reconciled, actorID); err != nil {
		return nil, err
	}
	return r.GetGroup(ctx, tx, id)
}

// GroupIDForTransaction devolve o grupo do lancamento (vazio quando avulso).
func (r *Repo) GroupIDForTransaction(ctx context.Context, tx pgx.Tx, id string) (string, error) {
	var groupID *string
	if err := tx.QueryRow(ctx, `SELECT group_id::text FROM financial_transactions WHERE id = $1::uuid`, id).Scan(&groupID); err != nil {
		return "", err
	}
	if groupID == nil {
		return "", nil
	}
	return *groupID, nil
}
