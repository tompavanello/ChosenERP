package finance

import (
	"context"
	"errors"
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
// A conta/filial/tipo sao derivados dos lancamentos (precisam ser iguais).
func (r *Repo) CreateGroup(ctx context.Context, tx pgx.Tx, tenantID, actorID string, in CreateGroupInput) (*TransactionGroup, error) {
	ids := uniqueIDs(in.TransactionIDs)
	if len(ids) < 2 {
		return nil, ErrGroupMinMembers
	}
	date := strings.TrimSpace(in.OccurredAt)
	if date == "" {
		date = time.Now().UTC().Format(dateLayout)
	}
	if _, err := time.ParseInLocation(dateLayout, date, time.UTC); err != nil {
		return nil, errors.New("data do grupo invalida")
	}

	rows, err := tx.Query(ctx, `
		SELECT id::text, tenant_id::text, branch_id::text, COALESCE(account_id::text,''),
		       type, amount::float8, voided_at, reconciled_at, group_id::text
		FROM financial_transactions
		WHERE id = ANY($1::uuid[])
		FOR UPDATE`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rowTenant, branch, account, kind string
	found := 0
	for rows.Next() {
		var id, tTenant, tBranch, tAccount, tType string
		var amount float64
		var voidedAt, reconciledAt *time.Time
		var groupID *string
		if err := rows.Scan(&id, &tTenant, &tBranch, &tAccount, &tType, &amount,
			&voidedAt, &reconciledAt, &groupID); err != nil {
			return nil, err
		}
		if found == 0 {
			rowTenant, branch, account, kind = tTenant, tBranch, tAccount, tType
		}
		if tTenant != rowTenant || tBranch != branch || tAccount != account || tType != kind {
			return nil, ErrGroupMixed
		}
		if voidedAt != nil {
			return nil, errors.New("lancamento estornado nao pode ser agrupado")
		}
		if reconciledAt != nil {
			return nil, ErrTransactionReconciled
		}
		if groupID != nil {
			return nil, ErrTransactionGrouped
		}
		if account == "" {
			return nil, ErrAccountRequired
		}
		found++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
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

	var groupID string
	err = tx.QueryRow(ctx, `
		INSERT INTO financial_transaction_groups
			(tenant_id, branch_id, account_id, type, occurred_at, description, created_by)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5::date, NULLIF($6,''), NULLIF($7,'')::uuid)
		RETURNING id::text`,
		tenantID, branch, account, kind, date, in.Description, actorID).Scan(&groupID)
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
		       g.status, g.reconciled_at, g.bank_entry_id::text, g.created_at,
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
		ORDER BY g.occurred_at DESC, g.created_at DESC`, from, to, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TransactionGroup{}
	for rows.Next() {
		var g TransactionGroup
		if err := rows.Scan(&g.ID, &g.BranchID, &g.AccountID, &g.AccountName,
			&g.Type, &g.OccurredAt, &g.Description, &g.Status, &g.ReconciledAt,
			&g.BankEntryID, &g.CreatedAt, &g.TotalAmount, &g.MemberCount); err != nil {
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
		       g.status, g.reconciled_at, g.bank_entry_id::text, g.created_at,
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
			&g.BankEntryID, &g.CreatedAt, &g.TotalAmount, &g.MemberCount)
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
	tag, err := tx.Exec(ctx, `
		UPDATE financial_transaction_groups
		SET occurred_at = COALESCE($2::date, occurred_at),
		    description = CASE WHEN $3::text IS NULL THEN description
		                       WHEN $3 = '' THEN NULL
		                       ELSE $3 END
		WHERE id = $1::uuid AND status = 'agrupado'`, id, date, description)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		var status string
		if err := tx.QueryRow(ctx, `SELECT status FROM financial_transaction_groups WHERE id=$1::uuid`, id).Scan(&status); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrGroupNotFound
			}
			return nil, err
		}
		if status == "conciliado" {
			return nil, ErrGroupReconciled
		}
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
	return nil
}

// AddGroupMembers anexa lancamentos ao grupo.
func (r *Repo) AddGroupMembers(ctx context.Context, tx pgx.Tx, id string, txIDs []string) (*TransactionGroup, error) {
	ids := uniqueIDs(txIDs)
	if len(ids) == 0 {
		return nil, errors.New("informe ao menos um lancamento")
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
