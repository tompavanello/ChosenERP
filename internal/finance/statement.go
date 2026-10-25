package finance

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// WeekRange descreve a faixa de datas de uma semana do Demonstrativo Mensal.
type WeekRange struct {
	Label string `json:"label"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// StatementLine e uma conta do plano com os totais por semana.
type StatementLine struct {
	CategoryID string    `json:"category_id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Weeks      []float64 `json:"weeks"` // 5 posicoes (semanas)
	Total      float64   `json:"total"`
}

// MonthlyStatement e o "Demonstrativo Mensal (Regime de Caixa)" do cliente:
// entradas e saidas por conta e por semana, com saldo inicial/final.
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

// parseMonth aceita "YYYY-MM" (vazio = mes corrente) e devolve o primeiro dia,
// o primeiro dia do mes seguinte e o rotulo.
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

// weekBounds devolve as semanas do mes com inicio no domingo e fim no sabado.
// A primeira semana pode comecar no mes anterior e a ultima terminar no mes
// seguinte, para que todas tenham exatamente 7 dias.
func weekBounds(first, next time.Time) []WeekRange {
	last := next.AddDate(0, 0, -1)
	start := first.AddDate(0, 0, -int(first.Weekday())) // domingo
	out := []WeekRange{}
	for w := start; !w.After(last); w = w.AddDate(0, 0, 7) {
		out = append(out, WeekRange{
			Label: "Semana " + strconv.Itoa(len(out)+1),
			From:  w.Format(dateLayout),
			To:    w.AddDate(0, 0, 6).Format(dateLayout),
		})
	}
	return out
}

// BuildMonthlyStatement monta o Demonstrativo Mensal do periodo.
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

	// Saldo inicial: liquido acumulado ANTES do mes (regime de caixa).
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0)::float8
		FROM financial_transactions
		WHERE occurred_at < $1::date AND voided_at IS NULL`, first).Scan(&st.OpeningBalance); err != nil {
		return st, err
	}

	// Contas ativas do plano, ordenadas pelo codigo (101, 102...). As semanas
	// cobrem o mes inteiro, comecando no domingo.
	catRows, err := tx.Query(ctx, `
		SELECT c.id::text, c.code, c.name, c.type
		FROM financial_categories c
		WHERE c.is_active
		ORDER BY c.type,
		         NULLIF(regexp_replace(c.code, '\D', '', 'g'), '')::int NULLS LAST,
		         c.code`)
	if err != nil {
		return st, err
	}
	byID := map[string]*StatementLine{}
	incomeLines := []*StatementLine{}
	expenseLines := []*StatementLine{}
	for catRows.Next() {
		l := &StatementLine{Weeks: make([]float64, len(st.Weeks))}
		var typ string
		if err := catRows.Scan(&l.CategoryID, &l.Code, &l.Name, &typ); err != nil {
			catRows.Close()
			return st, err
		}
		byID[l.CategoryID] = l
		if typ == "income" {
			incomeLines = append(incomeLines, l)
		} else {
			expenseLines = append(expenseLines, l)
		}
	}
	if err := catRows.Err(); err != nil {
		catRows.Close()
		return st, err
	}
	catRows.Close()

	// Totais por conta e por semana: o indice da semana e a distancia em dias
	// desde o domingo que abre o mes, dividida por 7.
	weekStart := first.AddDate(0, 0, -int(first.Weekday()))
	txRows, err := tx.Query(ctx, `
		SELECT t.category_id::text,
		       ((t.occurred_at::date - $1::date) / 7)::int AS wk,
		       COALESCE(SUM(t.amount), 0)::float8
		FROM financial_transactions t
		WHERE t.occurred_at >= $2::date AND t.occurred_at < $3::date
		  AND t.voided_at IS NULL
		  AND t.category_id IS NOT NULL
		GROUP BY 1, 2`, weekStart, first, next)
	if err != nil {
		return st, err
	}
	defer txRows.Close()
	for txRows.Next() {
		var categoryID string
		var wk int
		var amount float64
		if err := txRows.Scan(&categoryID, &wk, &amount); err != nil {
			return st, err
		}
		l := byID[categoryID]
		if l == nil || wk < 0 || wk >= len(l.Weeks) {
			continue
		}
		l.Weeks[wk] += amount
		l.Total += amount
	}
	if err := txRows.Err(); err != nil {
		return st, err
	}

	st.Income = []StatementLine{}
	st.Expense = []StatementLine{}
	for _, l := range incomeLines {
		st.Income = append(st.Income, *l)
		st.TotalIncome += l.Total
	}
	for _, l := range expenseLines {
		st.Expense = append(st.Expense, *l)
		st.TotalExpense += l.Total
	}

	st.ClosingBalance = st.OpeningBalance + st.TotalIncome - st.TotalExpense
	return st, nil
}
