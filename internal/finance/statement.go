package finance

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// WeekRange descreve a faixa de datas de uma semana do Demonstrativo Mensal.
type WeekRange struct {
	Label string `json:"label"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// StatementLine é uma conta do plano com os totais por semana.
type StatementLine struct {
	CategoryID string    `json:"category_id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Weeks      []float64 `json:"weeks"` // 5 posições (semanas)
	Total      float64   `json:"total"`
}

// MonthlyStatement é o "Demonstrativo Mensal (Regime de Caixa)" do cliente:
// entradas e saídas por conta e por semana, com saldo inicial/final.
type MonthlyStatement struct {
	Month          string          `json:"month"`
	From           string          `json:"from"`
	To             string          `json:"to"`
	Weeks          []WeekRange     `json:"weeks"`
	Income         []StatementLine `json:"income"`
	Expense        []StatementLine `json:"expense"`
	TotalIncome    float64         `json:"total_income"`
	TotalExpense   float64         `json:"total_expense"`
	OpeningBalance float64         `json:"opening_balance"`
	ClosingBalance float64         `json:"closing_balance"`
}

// parseMonth aceita "YYYY-MM" (vazio = mês corrente) e devolve o primeiro dia,
// o primeiro dia do mês seguinte e o rótulo.
func parseMonth(month string) (first, next time.Time, label string, err error) {
	now := time.Now().UTC()
	y, m := now.Year(), int(now.Month())
	if month != "" {
		t, e := time.Parse("2006-01", month)
		if e != nil {
			return first, next, label, e
		}
		y, m = t.Year(), int(t.Month())
	}
	first = time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
	next = first.AddDate(0, 1, 0)
	return first, next, first.Format("2006-01"), nil
}

// weekBounds devolve as 5 faixas de dias do mês (1–7, 8–14, 15–21, 22–28, 29–fim).
func weekBounds(first, next time.Time) []WeekRange {
	last := next.AddDate(0, 0, -1)
	d := func(n int) string {
		return time.Date(first.Year(), first.Month(), n, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	bounds := [][2]int{{1, 7}, {8, 14}, {15, 21}, {22, 28}, {29, last.Day()}}
	out := make([]WeekRange, 0, 5)
	for i, b := range bounds {
		out = append(out, WeekRange{
			Label: "Semana " + string(rune('1'+i)),
			From:  d(b[0]),
			To:    d(b[1]),
		})
	}
	return out
}

// BuildMonthlyStatement monta o Demonstrativo Mensal do período.
func (r *Repo) BuildMonthlyStatement(ctx context.Context, tx pgx.Tx, month string) (MonthlyStatement, error) {
	first, next, label, err := parseMonth(month)
	if err != nil {
		return MonthlyStatement{}, err
	}
	st := MonthlyStatement{
		Month: label,
		From:  first.Format("2006-01-02"),
		To:    next.AddDate(0, 0, -1).Format("2006-01-02"),
		Weeks: weekBounds(first, next),
	}

	// Saldo inicial: líquido acumulado ANTES do mês (regime de caixa).
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0)::float8
		FROM financial_transactions
		WHERE occurred_at < $1::date AND voided_at IS NULL`, first).Scan(&st.OpeningBalance); err != nil {
		return st, err
	}

	// Contas ativas do plano com os totais por semana (inclusive as sem movimento).
	rows, err := tx.Query(ctx, `
		SELECT c.id::text, c.code, c.name, c.type,
		       COALESCE(SUM(t.amount) FILTER (WHERE EXTRACT(DAY FROM t.occurred_at) <= 7), 0)::float8,
		       COALESCE(SUM(t.amount) FILTER (WHERE EXTRACT(DAY FROM t.occurred_at) BETWEEN 8 AND 14), 0)::float8,
		       COALESCE(SUM(t.amount) FILTER (WHERE EXTRACT(DAY FROM t.occurred_at) BETWEEN 15 AND 21), 0)::float8,
		       COALESCE(SUM(t.amount) FILTER (WHERE EXTRACT(DAY FROM t.occurred_at) BETWEEN 22 AND 28), 0)::float8,
		       COALESCE(SUM(t.amount) FILTER (WHERE EXTRACT(DAY FROM t.occurred_at) >= 29), 0)::float8,
		       COALESCE(SUM(t.amount), 0)::float8
		FROM financial_categories c
		LEFT JOIN financial_transactions t
		       ON t.category_id = c.id
		      AND t.occurred_at >= $1::date AND t.occurred_at < $2::date
		      AND t.voided_at IS NULL
		WHERE c.is_active
		GROUP BY c.id, c.code, c.name, c.type
		ORDER BY c.type,
		         NULLIF(regexp_replace(c.code, '\D', '', 'g'), '')::int NULLS LAST,
		         c.code`, first, next)
	if err != nil {
		return st, err
	}
	defer rows.Close()

	st.Income = []StatementLine{}
	st.Expense = []StatementLine{}
	for rows.Next() {
		var l StatementLine
		var typ string
		var w1, w2, w3, w4, w5, total float64
		if err := rows.Scan(&l.CategoryID, &l.Code, &l.Name, &typ, &w1, &w2, &w3, &w4, &w5, &total); err != nil {
			return st, err
		}
		l.Weeks = []float64{w1, w2, w3, w4, w5}
		l.Total = total
		if typ == "income" {
			st.Income = append(st.Income, l)
			st.TotalIncome += total
		} else {
			st.Expense = append(st.Expense, l)
			st.TotalExpense += total
		}
	}
	if err := rows.Err(); err != nil {
		return st, err
	}

	st.ClosingBalance = st.OpeningBalance + st.TotalIncome - st.TotalExpense
	return st, nil
}
