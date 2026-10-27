package finance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// AccountStatementLine e um lancamento do extrato com o saldo apos ele.
// Na visao "banco", um grupo aparece como UMA linha (IsGroup=true) e
// GroupMemberCount informa quantos lancamentos o compoem.
type AccountStatementLine struct {
	Transaction
	BalanceAfter     float64 `json:"balance_after"`
	IsGroup          bool    `json:"is_group,omitempty"`
	GroupMemberCount int     `json:"group_member_count,omitempty"`
}

// MarshalJSON preserva o formato plano do Transaction (occurred_at como data) e
// acrescenta os campos do extrato. Sem isto, o MarshalJSON promovido do
// Transaction descartaria os campos do extrato.
func (l AccountStatementLine) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(l.Transaction)
	if err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	b, err := json.Marshal(l.BalanceAfter)
	if err != nil {
		return nil, err
	}
	m["balance_after"] = b
	if l.IsGroup {
		m["is_group"] = json.RawMessage("true")
	}
	if l.GroupMemberCount > 0 {
		gc, err := json.Marshal(l.GroupMemberCount)
		if err != nil {
			return nil, err
		}
		m["group_member_count"] = gc
	}
	return json.Marshal(m)
}

// AccountStatementDay resume os movimentos de um dia.
type AccountStatementDay struct {
	Date           string  `json:"date"`
	Income         float64 `json:"income"`
	Expense        float64 `json:"expense"`
	Net            float64 `json:"net"`
	ClosingBalance float64 `json:"closing_balance"`
	Count          int     `json:"count"`
}

// AccountStatement e o extrato de uma conta em um periodo.
type AccountStatement struct {
	AccountID      string                 `json:"account_id"`
	AccountName    string                 `json:"account_name"`
	BranchID       *string                `json:"branch_id,omitempty"`
	InitialBalance float64                `json:"initial_balance"`
	From           string                 `json:"from"`
	To             string                 `json:"to"`
	Order          string                 `json:"order"`
	View           string                 `json:"view"`
	OpeningBalance float64                `json:"opening_balance"`
	TotalIncome    float64                `json:"total_income"`
	TotalExpense   float64                `json:"total_expense"`
	ClosingBalance float64                `json:"closing_balance"`
	Lines          []AccountStatementLine `json:"lines"`
	Days           []AccountStatementDay  `json:"days"`
}

// parseStatementDates resolve o periodo inclusivo do extrato.
func parseStatementDates(from, to string) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)

	var fromT, toT time.Time
	var err error
	switch {
	case from == "" && to == "":
		fromT = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		toT = time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	case from == "":
		if toT, err = time.ParseInLocation(dateLayout, to, time.UTC); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("data final invalida")
		}
		fromT = time.Date(toT.Year(), toT.Month(), 1, 0, 0, 0, 0, time.UTC)
	case to == "":
		if fromT, err = time.ParseInLocation(dateLayout, from, time.UTC); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("data inicial invalida")
		}
		toT = time.Date(fromT.Year(), fromT.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	default:
		if fromT, err = time.ParseInLocation(dateLayout, from, time.UTC); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("data inicial invalida")
		}
		if toT, err = time.ParseInLocation(dateLayout, to, time.UTC); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("data final invalida")
		}
	}
	if fromT.After(toT) {
		return time.Time{}, time.Time{}, fmt.Errorf("a data inicial nao pode ser maior que a final")
	}
	return fromT, toT, nil
}

// BuildAccountStatement monta o extrato de uma conta no periodo informado
// (datas inclusivas). order aceita "asc" (padrao) ou "desc". view aceita
// "detail" (padrao, um lancamento por linha) ou "bank" (um grupo por linha,
// na data do deposito).
func (r *Repo) BuildAccountStatement(ctx context.Context, tx pgx.Tx, accountID, from, to, order, view string) (*AccountStatement, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, ErrAccountRequired
	}
	fromT, toT, err := parseStatementDates(from, to)
	if err != nil {
		return nil, err
	}
	order = strings.ToLower(strings.TrimSpace(order))
	if order != "desc" {
		order = "asc"
	}
	view = strings.ToLower(strings.TrimSpace(view))
	if view != "bank" {
		view = "detail"
	}
	toExclusive := toT.AddDate(0, 0, 1)

	st := &AccountStatement{
		AccountID: accountID,
		From:      fromT.Format(dateLayout),
		To:        toT.Format(dateLayout),
		Order:     order,
		View:      view,
		Lines:     []AccountStatementLine{},
		Days:      []AccountStatementDay{},
	}
	if err := tx.QueryRow(ctx, `
		SELECT name, branch_id::text, initial_balance::float8
		FROM financial_accounts WHERE id = $1::uuid`, accountID).
		Scan(&st.AccountName, &st.BranchID, &st.InitialBalance); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	if view == "bank" {
		if err := r.buildBankStatement(ctx, tx, st, fromT, toExclusive); err != nil {
			return nil, err
		}
	} else {
		if err := r.buildDetailStatement(ctx, tx, st, fromT, toExclusive); err != nil {
			return nil, err
		}
	}

	st.Days = groupStatementDays(st.Lines)
	st.ClosingBalance = st.OpeningBalance + st.TotalIncome - st.TotalExpense

	if order == "desc" {
		for i, j := 0, len(st.Lines)-1; i < j; i, j = i+1, j-1 {
			st.Lines[i], st.Lines[j] = st.Lines[j], st.Lines[i]
		}
		for i, j := 0, len(st.Days)-1; i < j; i, j = i+1, j-1 {
			st.Days[i], st.Days[j] = st.Days[j], st.Days[i]
		}
	}
	return st, nil
}

func (r *Repo) buildDetailStatement(ctx context.Context, tx pgx.Tx, st *AccountStatement, fromT, toExclusive time.Time) error {
	var netBefore float64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0)::float8
		FROM financial_transactions
		WHERE account_id = $1::uuid AND voided_at IS NULL AND occurred_at < $2::date`,
		st.AccountID, fromT).Scan(&netBefore); err != nil {
		return err
	}
	st.OpeningBalance = st.InitialBalance + netBefore

	rows, err := tx.Query(ctx, `
		SELECT t.id::text, t.branch_id::text, t.category_id::text, c.name,
		       t.account_id::text, a.name,
		       t.supplier_id::text, s.name,
		       t.donor_member_id::text, t.benefactor_id::text,
		       COALESCE(t.donor_name, dm.full_name, bf.name),
		       t.type, t.amount::float8, t.currency, t.payment_method,
		       t.is_anonymous, t.description, t.receipt_issued, t.hash,
		       t.occurred_at, t.created_at, t.entry_seq,
		       (SELECT d.id::text FROM documents d
		         WHERE d.kind = 'receipt' AND d.content->>'tx_id' = t.id::text
		         ORDER BY d.created_at DESC LIMIT 1),
		       (SELECT d.document_ref FROM documents d
		         WHERE d.kind = 'receipt' AND d.content->>'tx_id' = t.id::text
		         ORDER BY d.created_at DESC LIMIT 1),
		       (SELECT d.qr_token FROM documents d
		         WHERE d.kind = 'receipt' AND d.content->>'tx_id' = t.id::text
		         ORDER BY d.created_at DESC LIMIT 1),
		       t.voided_at, t.void_reason,
		       (SELECT COUNT(*) FROM financial_attachments fa WHERE fa.transaction_id = t.id)::int,
		       (SELECT fa.file_url FROM financial_attachments fa
		         WHERE fa.transaction_id = t.id ORDER BY fa.created_at LIMIT 1),
		       t.due_date::text, t.paid_at, t.cost_center, t.group_id::text, g.status,
		       ($2::numeric + SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE -t.amount END)
		          OVER (ORDER BY t.occurred_at, t.entry_seq NULLS LAST, t.created_at, t.id
		                ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW))::float8
		FROM financial_transactions t
		LEFT JOIN financial_categories c ON c.id = t.category_id
		LEFT JOIN financial_accounts a ON a.id = t.account_id
		LEFT JOIN suppliers s ON s.id = t.supplier_id
		LEFT JOIN members dm ON dm.id = t.donor_member_id
		LEFT JOIN benefactors bf ON bf.id = t.benefactor_id
		LEFT JOIN financial_transaction_groups g ON g.id = t.group_id
		WHERE t.account_id = $1::uuid AND t.voided_at IS NULL
		  AND t.occurred_at >= $3::date AND t.occurred_at < $4::date
		ORDER BY t.occurred_at, t.entry_seq NULLS LAST, t.created_at, t.id`,
		st.AccountID, st.OpeningBalance, fromT, toExclusive)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var l AccountStatementLine
		if err := rows.Scan(&l.ID, &l.BranchID, &l.CategoryID, &l.CategoryName,
			&l.AccountID, &l.AccountName,
			&l.SupplierID, &l.SupplierName,
			&l.DonorMemberID, &l.BenefactorID, &l.DonorName,
			&l.Type, &l.Amount, &l.Currency, &l.PaymentMethod,
			&l.IsAnonymous, &l.Description, &l.ReceiptIssued, &l.Hash,
			&l.OccurredAt, &l.CreatedAt, &l.EntrySeq,
			&l.ReceiptID, &l.ReceiptRef, &l.ReceiptToken,
			&l.VoidedAt, &l.VoidReason, &l.AttachmentCount, &l.AttachmentURL,
			&l.DueDate, &l.PaidAt, &l.CostCenter, &l.GroupID, &l.GroupStatus,
			&l.BalanceAfter); err != nil {
			return err
		}
		if l.GroupID != nil {
			l.IsGroup = false
			l.GroupMemberCount = 0
		}
		if l.Type == "income" {
			st.TotalIncome += l.Amount
		} else {
			st.TotalExpense += l.Amount
		}
		st.Lines = append(st.Lines, l)
	}
	return rows.Err()
}

// buildBankStatement colapsa cada grupo em uma unica linha, usando a data do
// deposito (grupo) como data canonica da visao bancaria.
func (r *Repo) buildBankStatement(ctx context.Context, tx pgx.Tx, st *AccountStatement, fromT, toExclusive time.Time) error {
	var netBefore float64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE -t.amount END), 0)::float8
		FROM financial_transactions t
		LEFT JOIN financial_transaction_groups g ON g.id = t.group_id
		WHERE t.account_id = $1::uuid AND t.voided_at IS NULL
		  AND COALESCE(g.occurred_at, t.occurred_at) < $2::date`,
		st.AccountID, fromT).Scan(&netBefore); err != nil {
		return err
	}
	st.OpeningBalance = st.InitialBalance + netBefore

	rows, err := tx.Query(ctx, `
		WITH base AS (
		    SELECT COALESCE(t.group_id, t.id)::text AS line_key,
		           COALESCE(g.occurred_at, t.occurred_at) AS line_date,
		           COALESCE(g.description, t.description) AS line_description,
		           COALESCE(g.status, '') AS group_status,
		           t.group_id::text AS group_id,
		           t.type, t.amount::float8 AS amount, t.currency,
		           t.created_at,
		           t.entry_seq AS member_entry_seq,
		           g.entry_seq AS group_entry_seq,
		           t.category_id::text, c.name AS category_name,
		           COALESCE(t.donor_name, dm.full_name, bf.name) AS donor_name,
		           t.payment_method, t.branch_id::text,
		           (t.group_id IS NOT NULL) AS is_group
		    FROM financial_transactions t
		    LEFT JOIN financial_transaction_groups g ON g.id = t.group_id
		    LEFT JOIN financial_categories c ON c.id = t.category_id
		    LEFT JOIN members dm ON dm.id = t.donor_member_id
		    LEFT JOIN benefactors bf ON bf.id = t.benefactor_id
		    WHERE t.account_id = $1::uuid AND t.voided_at IS NULL
		      AND COALESCE(g.occurred_at, t.occurred_at) >= $2::date
		      AND COALESCE(g.occurred_at, t.occurred_at) < $3::date
		), lines AS (
		    SELECT line_key,
		           MAX(line_date) AS line_date,
		           COALESCE(MAX(line_description), 'Agrupamento de lancamentos') AS line_description,
		           MAX(group_status) AS group_status,
		           MAX(group_id) AS group_id,
		           MAX(type) AS type,
		           SUM(amount) AS amount,
		           MAX(currency) AS currency,
		           MIN(created_at) AS created_at,
		           CASE WHEN BOOL_OR(is_group) THEN MAX(group_entry_seq)
		                ELSE MIN(member_entry_seq) END AS entry_seq,
		           CASE WHEN BOOL_OR(is_group) THEN NULL ELSE MAX(category_id) END AS category_id,
		           CASE WHEN BOOL_OR(is_group) THEN NULL ELSE MAX(category_name) END AS category_name,
		           CASE WHEN BOOL_OR(is_group) THEN NULL ELSE MAX(donor_name) END AS donor_name,
		           CASE WHEN BOOL_OR(is_group) THEN NULL ELSE MAX(payment_method) END AS payment_method,
		           MAX(branch_id) AS branch_id,
		           BOOL_OR(is_group) AS is_group,
		           COUNT(*)::int AS member_count
		    FROM base
		    GROUP BY line_key
		), ordered AS (
		    SELECT *,
		           SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END)
		               OVER (ORDER BY line_date, entry_seq NULLS LAST, created_at, line_key) AS running
		    FROM lines
		)
		SELECT line_key, branch_id, category_id, category_name, type, amount, currency,
		       payment_method, false AS is_anonymous, line_description, false AS receipt_issued,
		       '' AS hash, line_date, created_at, entry_seq,
		       NULL::text AS receipt_id, NULL::text AS receipt_ref, NULL::text AS receipt_token,
		       NULL::timestamptz AS voided_at, NULL::text AS void_reason,
		       0 AS attachment_count, NULL::text AS attachment_url,
		       NULL::text AS due_date, NULL::timestamptz AS paid_at, NULL::text AS cost_center,
		       group_id, NULLIF(group_status, '') AS group_status,
		       is_group, member_count,
		       ($4::numeric + running)::float8 AS balance_after
		FROM ordered
		ORDER BY line_date, entry_seq NULLS LAST, created_at, line_key`,
		st.AccountID, fromT, toExclusive, st.OpeningBalance)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var l AccountStatementLine
		if err := rows.Scan(&l.ID, &l.BranchID, &l.CategoryID, &l.CategoryName,
			&l.Type, &l.Amount, &l.Currency, &l.PaymentMethod, &l.IsAnonymous,
			&l.Description, &l.ReceiptIssued, &l.Hash,
			&l.OccurredAt, &l.CreatedAt, &l.EntrySeq,
			&l.ReceiptID, &l.ReceiptRef, &l.ReceiptToken,
			&l.VoidedAt, &l.VoidReason, &l.AttachmentCount, &l.AttachmentURL,
			&l.DueDate, &l.PaidAt, &l.CostCenter,
			&l.GroupID, &l.GroupStatus, &l.IsGroup, &l.GroupMemberCount,
			&l.BalanceAfter); err != nil {
			return err
		}
		if l.Type == "income" {
			st.TotalIncome += l.Amount
		} else {
			st.TotalExpense += l.Amount
		}
		st.Lines = append(st.Lines, l)
	}
	return rows.Err()
}

func groupStatementDays(lines []AccountStatementLine) []AccountStatementDay {
	days := []AccountStatementDay{}
	index := map[string]int{}
	for _, l := range lines {
		day := l.OccurredAt.UTC().Format(dateLayout)
		i, ok := index[day]
		if !ok {
			i = len(days)
			index[day] = i
			days = append(days, AccountStatementDay{Date: day})
		}
		d := &days[i]
		if l.Type == "income" {
			d.Income += l.Amount
		} else {
			d.Expense += l.Amount
		}
		d.Net = d.Income - d.Expense
		d.ClosingBalance = l.BalanceAfter
		d.Count++
	}
	return days
}
