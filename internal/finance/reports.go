package finance

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// MonthlyPoint é um ponto da série de balancete mensal.
type MonthlyPoint struct {
	Month   string  `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Net     float64 `json:"net"`
}

// DRELine é uma linha do DRE (por categoria).
type DRELine struct {
	CategoryID string  `json:"category_id"`
	Category   string  `json:"category"`
	Type       string  `json:"type"`
	Total      float64 `json:"total"`
}

// Comparison compara o período atual com o imediatamente anterior de igual duração.
type Comparison struct {
	PrevIncome  float64 `json:"prev_income"`
	PrevExpense float64 `json:"prev_expense"`
	PrevNet     float64 `json:"prev_net"`
	DeltaPct    float64 `json:"delta_pct"`
}

// DRE é o demonstrativo de resultado do exercício (período).
type DRE struct {
	From       string      `json:"from"`
	To         string      `json:"to"`
	Income     float64     `json:"income"`
	Expense    float64     `json:"expense"`
	Net        float64     `json:"net"`
	Lines      []DRELine   `json:"lines"`
	Comparison *Comparison `json:"comparison,omitempty"`
}

const dateLayout = "2006-01-02"

// resolvePeriod aplica padrão (mês corrente) e converte em limites [from, to].
func resolvePeriod(from, to string) (string, string) {
	now := time.Now().UTC()
	f := from
	t := to
	if f == "" {
		f = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format(dateLayout)
	}
	if t == "" {
		t = time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Format(dateLayout)
	}
	return f, t
}

// MonthlySeries retorna o balancete agregado por mês dentro do período.
func (r *Repo) MonthlySeries(ctx context.Context, tx pgx.Tx, from, to string) ([]MonthlyPoint, error) {
	from, to = resolvePeriod(from, to)
	rows, err := tx.Query(ctx, `
		SELECT to_char(date_trunc('month', occurred_at), 'YYYY-MM') AS m,
		       COALESCE(SUM(amount) FILTER (WHERE type='income'),0),
		       COALESCE(SUM(amount) FILTER (WHERE type='expense'),0)
		FROM financial_transactions
		WHERE occurred_at >= NULLIF($1,'')::date
		  AND occurred_at < (NULLIF($2,'')::date + 1)
		GROUP BY 1 ORDER BY 1`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MonthlyPoint{}
	for rows.Next() {
		var p MonthlyPoint
		if err := rows.Scan(&p.Month, &p.Income, &p.Expense); err != nil {
			return nil, err
		}
		p.Net = p.Income - p.Expense
		out = append(out, p)
	}
	return out, rows.Err()
}

// BuildDRE monta o DRE do período com linhas por categoria e comparativo.
func (r *Repo) BuildDRE(ctx context.Context, tx pgx.Tx, from, to string) (DRE, error) {
	from, to = resolvePeriod(from, to)
	var dre DRE
	dre.From = from
	dre.To = to

	err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount) FILTER (WHERE type='income'),0),
		       COALESCE(SUM(amount) FILTER (WHERE type='expense'),0)
		FROM financial_transactions
		WHERE occurred_at >= NULLIF($1,'')::date
		  AND occurred_at < (NULLIF($2,'')::date + 1)`, from, to).
		Scan(&dre.Income, &dre.Expense)
	if err != nil {
		return dre, err
	}
	dre.Net = dre.Income - dre.Expense

	rows, err := tx.Query(ctx, `
		SELECT COALESCE(t.category_id::text,''), COALESCE(c.name,'Sem categoria'), t.type, SUM(t.amount)::float8
		FROM financial_transactions t
		LEFT JOIN financial_categories c ON c.id = t.category_id
		WHERE t.occurred_at >= NULLIF($1,'')::date
		  AND t.occurred_at < (NULLIF($2,'')::date + 1)
		GROUP BY 1,2,3 ORDER BY 3, 4 DESC`, from, to)
	if err != nil {
		return dre, err
	}
	defer rows.Close()
	dre.Lines = []DRELine{}
	for rows.Next() {
		var l DRELine
		if err := rows.Scan(&l.CategoryID, &l.Category, &l.Type, &l.Total); err != nil {
			return dre, err
		}
		dre.Lines = append(dre.Lines, l)
	}
	if err := rows.Err(); err != nil {
		return dre, err
	}

	// Período anterior de igual duração
	var prevIncome, prevExpense float64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount) FILTER (WHERE type='income'),0),
		       COALESCE(SUM(amount) FILTER (WHERE type='expense'),0)
		FROM financial_transactions
		WHERE occurred_at >= (NULLIF($1,'')::date - (NULLIF($2,'')::date - NULLIF($1,'')::date + 1))
		  AND occurred_at < NULLIF($1,'')::date`, from, to).
		Scan(&prevIncome, &prevExpense)
	if err != nil {
		return dre, err
	}
	cmp := Comparison{PrevIncome: prevIncome, PrevExpense: prevExpense, PrevNet: prevIncome - prevExpense}
	if cmp.PrevNet != 0 {
		cmp.DeltaPct = ((dre.Net - cmp.PrevNet) / abs(cmp.PrevNet)) * 100
	}
	dre.Comparison = &cmp
	return dre, nil
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
