package finance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrReconciliationClosed indica conciliacao ja concluida (periodo travado).
var ErrReconciliationClosed = errors.New("conciliacao concluida: periodo travado")

// Reconciliation e a conciliacao de um periodo de uma filial. Depois de
// conciliada, o periodo e travado no banco (ver trigger fin_tx_period_lock).
type Reconciliation struct {
	ID           string  `json:"id"`
	BranchID     string  `json:"branch_id"`
	Title        string  `json:"title"`
	PeriodStart  string  `json:"period_start"`
	PeriodEnd    string  `json:"period_end"`
	Status       string  `json:"status"`
	Notes        *string `json:"notes,omitempty"`
	ReconciledAt *string `json:"reconciled_at,omitempty"`
	CreatedAt    string  `json:"created_at"`

	TotalItems   int     `json:"total_items"`
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
}

// ReconciliationItem e um lancamento do periodo da conciliacao (data efetiva
// devolvida como YYYY-MM-DD, sem deslocamento de fuso).
type ReconciliationItem struct {
	ID           string  `json:"id"`
	OccurredAt   string  `json:"occurred_at"`
	Type         string  `json:"type"`
	Amount       float64 `json:"amount"`
	Description  *string `json:"description,omitempty"`
	CategoryName *string `json:"category_name,omitempty"`
	AccountName  *string `json:"account_name,omitempty"`
}

type CreateReconciliationInput struct {
	Title       string  `json:"title"`
	PeriodStart string  `json:"period_start"`
	PeriodEnd   string  `json:"period_end"`
	Notes       *string `json:"notes"`
}

const reconciliationCols = `fr.id::text, fr.branch_id::text, fr.title,
	to_char(fr.period_start,'YYYY-MM-DD'), to_char(fr.period_end,'YYYY-MM-DD'),
	fr.status, fr.notes, to_char(fr.reconciled_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
	to_char(fr.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
	COALESCE(cnt.n,0)::int, COALESCE(cnt.income,0)::float8, COALESCE(cnt.expense,0)::float8`

const reconciliationJoin = `
	FROM financial_reconciliations fr
	LEFT JOIN LATERAL (
	    SELECT count(*) AS n,
	           SUM(t.amount) FILTER (WHERE t.type='income')  AS income,
	           SUM(t.amount) FILTER (WHERE t.type='expense') AS expense
	    FROM financial_transactions t
	    WHERE t.tenant_id = fr.tenant_id
	      AND t.branch_id = fr.branch_id
	      AND t.voided_at IS NULL
	      AND t.occurred_at::date BETWEEN fr.period_start AND fr.period_end
	) cnt ON true`

func scanReconciliation(row pgx.Row) (*Reconciliation, error) {
	var rec Reconciliation
	var reconciled *string
	err := row.Scan(&rec.ID, &rec.BranchID, &rec.Title, &rec.PeriodStart, &rec.PeriodEnd,
		&rec.Status, &rec.Notes, &reconciled, &rec.CreatedAt,
		&rec.TotalItems, &rec.TotalIncome, &rec.TotalExpense)
	if err != nil {
		return nil, err
	}
	rec.ReconciledAt = reconciled
	return &rec, nil
}

// ListReconciliations lista as conciliacoes visiveis no escopo.
func (r *Repo) ListReconciliations(ctx context.Context, tx pgx.Tx) ([]Reconciliation, error) {
	rows, err := tx.Query(ctx, `SELECT `+reconciliationCols+reconciliationJoin+`
		ORDER BY fr.period_start DESC, fr.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Reconciliation{}
	for rows.Next() {
		rec, err := scanReconciliation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

// GetReconciliation devolve uma conciliacao pelo id.
func (r *Repo) GetReconciliation(ctx context.Context, tx pgx.Tx, id string) (*Reconciliation, error) {
	return scanReconciliation(tx.QueryRow(ctx, `SELECT `+reconciliationCols+reconciliationJoin+`
		WHERE fr.id = $1::uuid`, id))
}

// CreateReconciliation abre uma conciliacao para o periodo da filial. O guard
// no banco recusa periodos sobrepostos na mesma filial.
func (r *Repo) CreateReconciliation(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, in CreateReconciliationInput) (*Reconciliation, error) {
	title := in.Title
	if title == "" {
		title = "Conciliacao financeira"
	}
	if in.PeriodStart == "" || in.PeriodEnd == "" {
		return nil, errors.New("period_start e period_end sao obrigatorios")
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO financial_reconciliations
			(tenant_id, branch_id, title, period_start, period_end, notes, created_by)
		VALUES ($1, $2::uuid, $3, $4::date, $5::date, $6, NULLIF($7,'')::uuid)
		RETURNING id::text`,
		tenantID, branchID, title, in.PeriodStart, in.PeriodEnd, in.Notes, actorID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetReconciliation(ctx, tx, id)
}

// ListReconciliationItems lista os lancamentos do periodo (nao estornados).
func (r *Repo) ListReconciliationItems(ctx context.Context, tx pgx.Tx, id string) ([]ReconciliationItem, error) {
	rows, err := tx.Query(ctx, `
		SELECT t.id::text, to_char(t.occurred_at,'YYYY-MM-DD'), t.type, t.amount::float8,
		       t.description, c.name, a.name
		FROM financial_reconciliations fr
		JOIN financial_transactions t
		  ON t.tenant_id = fr.tenant_id
		 AND t.branch_id = fr.branch_id
		 AND t.voided_at IS NULL
		 AND t.occurred_at::date BETWEEN fr.period_start AND fr.period_end
		LEFT JOIN financial_categories c ON c.id = t.category_id
		LEFT JOIN financial_accounts a ON a.id = t.account_id
		WHERE fr.id = $1::uuid
		ORDER BY t.occurred_at, t.created_at, t.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReconciliationItem{}
	for rows.Next() {
		var it ReconciliationItem
		if err := rows.Scan(&it.ID, &it.OccurredAt, &it.Type, &it.Amount,
			&it.Description, &it.CategoryName, &it.AccountName); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// Conciliate conclui a conciliacao e TRAVA o periodo. O hash sela o conteudo
// (periodo + lancamentos) para a trilha de integridade.
func (r *Repo) Conciliate(ctx context.Context, tx pgx.Tx, id, actorID string) (*Reconciliation, error) {
	rec, err := r.GetReconciliation(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if rec.Status == "conciliada" {
		return nil, ErrReconciliationClosed
	}
	items, err := r.ListReconciliationItems(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s\n", rec.ID, rec.BranchID, rec.PeriodStart, rec.PeriodEnd)
	for _, it := range items {
		fmt.Fprintf(h, "%s|%s|%.2f\n", it.ID, it.OccurredAt, it.Amount)
	}
	sum := hex.EncodeToString(h.Sum(nil))

	tag, err := tx.Exec(ctx, `
		UPDATE financial_reconciliations
		SET status = 'conciliada', reconciled_at = now(),
		    reconciled_by = NULLIF($2,'')::uuid, signature_hash = $3, updated_at = now()
		WHERE id = $1::uuid AND status = 'aberta'`, id, actorID, sum)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrReconciliationClosed
	}
	return r.GetReconciliation(ctx, tx, id)
}

// DeleteReconciliation remove a conciliacao (destrava o periodo). Acao de
// manutencao restrita a Sede na UI.
func (r *Repo) DeleteReconciliation(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM financial_reconciliations WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
