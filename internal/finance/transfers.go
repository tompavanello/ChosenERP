package finance

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Transfer representa um repasse entre filiais (split de contribuicao/recurso).
type Transfer struct {
	ID            string    `json:"id"`
	BranchID      string    `json:"branch_id"`
	FromBranchID  string    `json:"from_branch_id"`
	FromBranch    string    `json:"from_branch"`
	ToBranchID    string    `json:"to_branch_id"`
	ToBranch      string    `json:"to_branch"`
	TransactionID *string   `json:"transaction_id,omitempty"`
	Amount        float64   `json:"amount"`
	RuleName      *string   `json:"rule_name,omitempty"`
	ExecutedAt    time.Time `json:"executed_at"`
}

type CreateTransferInput struct {
	FromBranchID  string  `json:"from_branch_id"`
	ToBranchID    string  `json:"to_branch_id"`
	Amount        float64 `json:"amount"`
	RuleName      *string `json:"rule_name"`
	TransactionID *string `json:"transaction_id"`
}

// CreateTransfer registra um repasse entre filiais no escopo RLS da sessao.
func (r *Repo) CreateTransfer(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateTransferInput) (*Transfer, error) {
	if in.Amount <= 0 {
		return nil, pgx.ErrNoRows
	}
	var t Transfer
	err := tx.QueryRow(ctx, `
		INSERT INTO transfers
			(tenant_id, from_branch_id, to_branch_id, financial_transaction_id, amount, rule_name)
		VALUES
			($1, $2::uuid, $3::uuid, NULLIF($4,'')::uuid, $5, $6)
		RETURNING id::text, from_branch_id::text, to_branch_id::text,
		          financial_transaction_id::text, amount::float8, rule_name, executed_at`,
		tenantID, in.FromBranchID, in.ToBranchID, nullIfEmpty(in.TransactionID), in.Amount, in.RuleName).
		Scan(&t.ID, &t.FromBranchID, &t.ToBranchID, &t.TransactionID, &t.Amount, &t.RuleName, &t.ExecutedAt)
	if err != nil {
		return nil, err
	}
	t.BranchID = branchID
	return &t, nil
}

// ListTransfers retorna os repasses visiveis no escopo (origem/destino da sessao).
func (r *Repo) ListTransfers(ctx context.Context, tx pgx.Tx) ([]Transfer, error) {
	rows, err := tx.Query(ctx, `
		SELECT tr.id::text, tr.from_branch_id::text, bf.name,
		       tr.to_branch_id::text, bt.name,
		       tr.financial_transaction_id::text, tr.amount::float8, tr.rule_name, tr.executed_at
		FROM transfers tr
		JOIN branches bf ON bf.id = tr.from_branch_id
		JOIN branches bt ON bt.id = tr.to_branch_id
		ORDER BY tr.executed_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Transfer{}
	for rows.Next() {
		var t Transfer
		if err := rows.Scan(&t.ID, &t.FromBranchID, &t.FromBranch, &t.ToBranchID, &t.ToBranch,
			&t.TransactionID, &t.Amount, &t.RuleName, &t.ExecutedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
