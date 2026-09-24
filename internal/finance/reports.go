package finance

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
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
		  AND voided_at IS NULL
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
		  AND occurred_at < (NULLIF($2,'')::date + 1)
		  AND voided_at IS NULL`, from, to).
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
		  AND t.voided_at IS NULL
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
		  AND occurred_at < NULLIF($1,'')::date
		  AND voided_at IS NULL`, from, to).
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

// BalanceteCSV monta o CSV (separador ';') do balancete mensal.
func BalanceteCSV(points []MonthlyPoint) (string, error) {
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	_ = w.Write([]string{"Mês", "Entradas", "Saídas", "Saldo"})
	for _, p := range points {
		_ = w.Write([]string{p.Month, fmtFloat(p.Income), fmtFloat(p.Expense), fmtFloat(p.Net)})
	}
	w.Flush()
	return buf.String(), w.Error()
}

// DRECSV monta o CSV do DRE por categoria.
func DRECSV(d DRE) (string, error) {
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	_ = w.Write([]string{"Indicador/Regra", "Tipo", "Categoria", "Total"})
	_ = w.Write([]string{"Período", "", d.From, d.To})
	for _, l := range d.Lines {
		_ = w.Write([]string{"Categoria", l.Type, l.Category, fmtFloat(l.Total)})
	}
	_ = w.Write([]string{"Entradas", "", "", fmtFloat(d.Income)})
	_ = w.Write([]string{"Saídas", "", "", fmtFloat(d.Expense)})
	_ = w.Write([]string{"Resultado", "", "", fmtFloat(d.Net)})
	if d.Comparison != nil {
		_ = w.Write([]string{"Comparativo (período anterior)", "", "", fmtFloat(d.Comparison.DeltaPct) + "%"})
	}
	w.Flush()
	return buf.String(), w.Error()
}

// DREPrintHTML devolve uma versão de impressão (HTML) do DRE, renderizável em PDF.
func DREPrintHTML(d DRE) string {
	brl := func(v float64) string { return fmt.Sprintf("R$ %.2f", v) }
	var body string
	for _, l := range d.Lines {
		body += "<tr><td>" + htmlEsc(l.Type) + "</td><td>" + htmlEsc(l.Category) + "</td><td class=r>" + brl(l.Total) + "</td></tr>"
	}
	if body == "" {
		body = "<tr><td colspan=3 class=muted>Sem lançamentos no período.</td></tr>"
	}
	return `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><title>DRE</title>
<style>body{font-family:ui-sans-serif,system-ui,sans-serif;padding:32px;color:#0b1020}table{width:100%;border-collapse:collapse;font-size:13px}th,td{text-align:left;padding:6px 8px;border-bottom:1px solid #e4e7ee}.r{text-align:right}.muted{color:#9ca3af}.tot{font-weight:600}.head{margin-bottom:16px}.head h1{font-size:18px;margin:0}.head p{margin:2px 0;color:#6b7280}@media print{body{padding:0}}</style></head><body>
<div class="head"><h1>DRE — Demonstração do Resultado</h1><p>Período: ` + htmlEsc(d.From) + ` a ` + htmlEsc(d.To) + `</p></div>
<table><thead><tr><th>Tipo</th><th>Categoria</th><th class=r>Total</th></tr></thead><tbody>` + body + `</tbody></table>
<p>Entradas: ` + brl(d.Income) + ` · Saídas: ` + brl(d.Expense) + ` · Resultado: ` + brl(d.Net) + `</p>
</body></html>`
}

func htmlEsc(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '&':
			b.WriteString("&amp;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}
