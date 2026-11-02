// Package insights transforma os numeros da igreja em leituras acionaveis.
//
// As regras sao determin�sticas (sem IA): cada indicador tem um benchmark
// configuravel e gera um cartao com severidade, mensagem e acao. A narrativa
// por IA e opcional e recebe somente os cartoes agregados (nunca dados pessoais).
package insights

import (
	"context"
	"fmt"
	"time"

	"chosenerp/internal/finance"
	"github.com/jackc/pgx/v5"
)

// Severity classifica a urgencia do insight.
type Severity string

const (
	Good     Severity = "good"
	Info     Severity = "info"
	Warn     Severity = "warn"
	Critical Severity = "critical"
)

// Action aponta para a tela onde o problema pode ser resolvido.
type Action struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

// Insight e um cartao de leitura sobre os dados.
type Insight struct {
	ID        string   `json:"id"`
	Severity  Severity `json:"severity"`
	Title     string   `json:"title"`
	Message   string   `json:"message"`
	Metric    string   `json:"metric,omitempty"`
	Benchmark string   `json:"benchmark,omitempty"`
	Action    *Action  `json:"action,omitempty"`
}

// Summary expoe os numeros que alimentaram as regras (tambem usados pela IA).
type Summary struct {
	Income            float64 `json:"income"`
	Expense           float64 `json:"expense"`
	Net               float64 `json:"net"`
	Cash              float64 `json:"cash"`
	MonthlyExpense    float64 `json:"monthly_expense"`
	RunwayMonths      float64 `json:"runway_months"`
	ExpenseRatio      float64 `json:"expense_ratio"`
	Donors            int     `json:"donors"`
	ActiveMembers     int     `json:"active_members"`
	TotalMembers      int     `json:"total_members"`
	Top10Share        float64 `json:"top10_share"`
	RecurringAmount   float64 `json:"recurring_amount"`
	RecurringCount    int     `json:"recurring_count"`
	Visitors          int     `json:"visitors"`
	ConvertedVisitors int     `json:"converted_visitors"`
	GroupMembers      int     `json:"group_members"`
	Events            int     `json:"events"`
	Participants      int     `json:"participants"`
	MissingContact    int     `json:"missing_contact"`
	ZeroAttendance    int     `json:"zero_attendance"`
	UpcomingNoRoster  int     `json:"upcoming_without_roster"`
	PendingAssignment int     `json:"pending_assignments"`
	LapsedGivers      int     `json:"lapsed_givers"`
}

// Report e a resposta de GET /insights.
type Report struct {
	Scope       string    `json:"scope"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	GeneratedAt time.Time `json:"generated_at"`
	Summary     Summary   `json:"summary"`
	Insights    []Insight `json:"insights"`
}

// Repo e sem estado: as consultas rodam na transacao com o escopo do tenant.
type Repo struct{}

// Benchmarks iniciais (sobrescreva em uma tabela de settings quando necessario).
const (
	runwayMin       = 3.0
	runwayGood      = 6.0
	expenseRatioOK  = 95.0
	concentrationOK = 40.0
	concentrationHi = 60.0
	recurringGood   = 50.0
	donorRatioMin   = 25.0
	visitorConvOK   = 20.0
	visitorConvGood = 40.0
	groupPartMin    = 40.0
	contactGapMax   = 10.0
)

// Build calcula o retrato do periodo e aplica as regras.
func (r *Repo) Build(ctx context.Context, tx pgx.Tx, scope, from, to string) (Report, error) {
	var s Summary
	if err := r.financeSnapshot(ctx, tx, from, to, &s); err != nil {
		return Report{}, err
	}
	if err := r.peopleSnapshot(ctx, tx, from, to, &s); err != nil {
		return Report{}, err
	}
	if err := r.operationsSnapshot(ctx, tx, from, to, &s); err != nil {
		return Report{}, err
	}

	rep := Report{
		Scope:       scope,
		From:        from,
		To:          to,
		GeneratedAt: time.Now(),
		Summary:     s,
		Insights:    []Insight{},
	}
	all := append(r.financeInsights(s), r.peopleInsights(s)...)
	all = append(all, r.operationsInsights(s)...)

	if scope != "" && scope != "all" {
		filtered := make([]Insight, 0, len(all))
		for _, it := range all {
			if scopeOf(it.ID) == scope {
				filtered = append(filtered, it)
			}
		}
		all = filtered
	}
	rep.Insights = all
	return rep, nil
}

func scopeOf(id string) string {
	switch {
	case len(id) >= 3 && id[:3] == "fin":
		return "finance"
	case len(id) >= 3 && id[:3] == "peo":
		return "people"
	case len(id) >= 3 && id[:3] == "ope":
		return "operations"
	default:
		return "all"
	}
}

func (r *Repo) financeSnapshot(ctx context.Context, tx pgx.Tx, from, to string, s *Summary) error {
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount * sign) FILTER (WHERE type='income'),0)::float8,
		       COALESCE(SUM(amount * sign) FILTER (WHERE type='expense'),0)::float8,
		       COUNT(DISTINCT donor_member_id) FILTER (WHERE type='income' AND donor_member_id IS NOT NULL)::int
		FROM financial_transactions
		WHERE occurred_at >= $1::date AND occurred_at < ($2::date + 1)
		  AND voided_at IS NULL AND account_transfer_id IS NULL`, from, to).
		Scan(&s.Income, &s.Expense, &s.Donors)
	if err != nil {
		return err
	}
	s.Net = s.Income - s.Expense
	if s.Income > 0 {
		s.ExpenseRatio = (s.Expense / s.Income) * 100
	}

	// Top10Share = percentual da arrecadacao vindo dos 10 maiores doadores.
	var topTotal, grand float64
	if err := tx.QueryRow(ctx, `
		WITH donor AS (
			SELECT donor_member_id, SUM(amount * sign) AS total
			FROM financial_transactions
			WHERE type='income' AND sign=1 AND donor_member_id IS NOT NULL
			  AND occurred_at >= $1::date AND occurred_at < ($2::date + 1)
			  AND voided_at IS NULL AND account_transfer_id IS NULL
			GROUP BY donor_member_id
		), ranked AS (
			SELECT total, ROW_NUMBER() OVER (ORDER BY total DESC) AS rn,
			       SUM(total) OVER () AS grand
			FROM donor
		)
		SELECT COALESCE(SUM(total) FILTER (WHERE rn <= 10),0)::float8,
		       COALESCE(MAX(grand),0)::float8
		FROM ranked`, from, to).Scan(&topTotal, &grand); err != nil {
		return err
	}
	if grand > 0 {
		s.Top10Share = (topTotal / grand) * 100
	}

	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount * sign),0)::float8 / 6.0
		FROM financial_transactions
		WHERE type='expense' AND occurred_at >= now() - interval '6 months'
		  AND voided_at IS NULL AND account_transfer_id IS NULL`).Scan(&s.MonthlyExpense); err != nil {
		return err
	}
	fin := &finance.Repo{}
	bal, err := fin.SumBalance(ctx, tx, "")
	if err != nil {
		return err
	}
	s.Cash = bal.Net
	if s.MonthlyExpense > 0 {
		s.RunwayMonths = s.Cash / s.MonthlyExpense
	}

	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount),0)::float8, COUNT(*)::int
		FROM recurring_donations WHERE is_active`).Scan(&s.RecurringAmount, &s.RecurringCount); err != nil {
		return err
	}

	if err := tx.QueryRow(ctx, `
		WITH contrib AS (
			SELECT donor_member_id AS member_id, MAX(occurred_at) AS last_at
			FROM financial_transactions
			WHERE type='income' AND sign=1 AND voided_at IS NULL AND donor_member_id IS NOT NULL
			  AND occurred_at >= now() - interval '6 months'
			GROUP BY donor_member_id
		)
		SELECT COUNT(*)::int FROM contrib
		WHERE last_at < now() - interval '1 month'`).Scan(&s.LapsedGivers); err != nil {
		return err
	}
	return nil
}

func (r *Repo) peopleSnapshot(ctx context.Context, tx pgx.Tx, from, to string, s *Summary) error {
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int,
		       COUNT(*) FILTER (WHERE membership_status IN ('active','member'))::int,
		       COUNT(*) FILTER (WHERE email IS NULL AND phone IS NULL)::int
		FROM members`).Scan(&s.TotalMembers, &s.ActiveMembers, &s.MissingContact); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int,
		       COUNT(*) FILTER (WHERE converted_to_member_id IS NOT NULL OR journey_stage='converted')::int
		FROM visitors
		WHERE created_at >= $1::date AND created_at < ($2::date + 1)`, from, to).
		Scan(&s.Visitors, &s.ConvertedVisitors); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(DISTINCT gm.member_id)::int
		FROM group_members gm
		JOIN members m ON m.id = gm.member_id
		WHERE m.membership_status IN ('active','member')`).Scan(&s.GroupMembers); err != nil {
		return err
	}
	return nil
}

func (r *Repo) operationsSnapshot(ctx context.Context, tx pgx.Tx, from, to string, s *Summary) error {
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int, COALESCE(SUM(participants_count),0)::int,
		       COUNT(*) FILTER (WHERE participants_count = 0 AND starts_at < now())::int
		FROM church_events
		WHERE starts_at >= $1::date AND starts_at < ($2::date + 1)`, from, to).
		Scan(&s.Events, &s.Participants, &s.ZeroAttendance); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM church_events e
		WHERE e.starts_at >= now() AND e.starts_at < now() + interval '30 days'
		  AND NOT EXISTS (
		    SELECT 1 FROM rosters r WHERE r.event_id = e.id AND r.status <> 'cancelada'
		  )`).Scan(&s.UpcomingNoRoster); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM roster_assignments ra
		JOIN rosters r ON r.id = ra.roster_id
		WHERE ra.status = 'convidado'
		  AND r.status <> 'cancelada'
		  AND r.starts_at >= now() AND r.starts_at < now() + interval '30 days'`).
		Scan(&s.PendingAssignment); err != nil {
		return err
	}
	return nil
}

func (r *Repo) financeInsights(s Summary) []Insight {
	out := []Insight{}

	if s.Income == 0 && s.Expense == 0 {
		return append(out, Insight{
			ID: "fin_empty", Severity: Info, Title: "Sem movimento financeiro no periodo",
			Message: "Nenhuma entrada ou saida registrada. Confira se os lancamentos estao no periodo e na conta correta.",
			Action:  &Action{Label: "Abrir financeiro", Href: "/dashboard/finance"},
		})
	}

	switch {
	case s.RunwayMonths > 0 && s.RunwayMonths < runwayMin:
		out = append(out, Insight{
			ID: "fin_runway", Severity: Critical, Title: "Reserva de caixa curta",
			Message: fmt.Sprintf("O caixa cobre cerca de %.1f mes(es) de despesa. O ideal e manter de 3 a 6 meses.", s.RunwayMonths),
			Metric:  fmt.Sprintf("%.1f meses", s.RunwayMonths), Benchmark: "3 a 6 meses",
			Action: &Action{Label: "Ver balancete", Href: "/dashboard/reports/balance"},
		})
	case s.RunwayMonths >= runwayGood:
		out = append(out, Insight{
			ID: "fin_runway", Severity: Good, Title: "Reserva de caixa saudavel",
			Message: fmt.Sprintf("O caixa cobre cerca de %.1f meses de despesa, dentro da referencia de 3 a 6 meses.", s.RunwayMonths),
			Metric:  fmt.Sprintf("%.1f meses", s.RunwayMonths), Benchmark: "3 a 6 meses",
		})
	}

	switch {
	case s.ExpenseRatio > 100:
		out = append(out, Insight{
			ID: "fin_expense_ratio", Severity: Critical, Title: "Despesas acima das entradas",
			Message: fmt.Sprintf("As saidas consumiram %.0f%% das entradas do periodo. E preciso rever despesas ou reforcar a arrecadacao.", s.ExpenseRatio),
			Metric:  fmt.Sprintf("%.0f%%", s.ExpenseRatio), Benchmark: "ate 95%",
			Action: &Action{Label: "Ver DRE", Href: "/dashboard/reports/dre"},
		})
	case s.ExpenseRatio > expenseRatioOK:
		out = append(out, Insight{
			ID: "fin_expense_ratio", Severity: Warn, Title: "Margem apertada",
			Message: fmt.Sprintf("As saidas consumiram %.0f%% das entradas. Pequenas variacoes ja viram deficit.", s.ExpenseRatio),
			Metric:  fmt.Sprintf("%.0f%%", s.ExpenseRatio), Benchmark: "ate 95%",
			Action: &Action{Label: "Ver DRE", Href: "/dashboard/reports/dre"},
		})
	}

	switch {
	case s.Top10Share > concentrationHi:
		out = append(out, Insight{
			ID: "fin_concentration", Severity: Warn, Title: "Arrecadacao concentrada",
			Message: fmt.Sprintf("Os 10 maiores doadores respondem por %.0f%% da arrecadacao. Ha risco se um deles parar.", s.Top10Share),
			Metric:  fmt.Sprintf("%.0f%%", s.Top10Share), Benchmark: "abaixo de 40%",
			Action: &Action{Label: "Ver concentracao", Href: "/dashboard/reports/contribution-drop"},
		})
	case s.Top10Share > concentrationOK:
		out = append(out, Insight{
			ID: "fin_concentration", Severity: Info, Title: "Concentracao a acompanhar",
			Message: fmt.Sprintf("Os 10 maiores doadores somam %.0f%% da arrecadacao. Vale ampliar a base de contribuintes.", s.Top10Share),
			Metric:  fmt.Sprintf("%.0f%%", s.Top10Share), Benchmark: "abaixo de 40%",
		})
	}

	if s.ActiveMembers > 0 && s.Donors > 0 {
		ratio := (float64(s.Donors) / float64(s.ActiveMembers)) * 100
		if ratio < donorRatioMin {
			out = append(out, Insight{
				ID: "fin_donor_ratio", Severity: Warn, Title: "Poucos membros contribuindo",
				Message: fmt.Sprintf("Apenas %.0f%% dos membros ativos contribuiram no periodo (%d de %d).", ratio, s.Donors, s.ActiveMembers),
				Metric:  fmt.Sprintf("%.0f%%", ratio), Benchmark: "acima de 25%",
				Action: &Action{Label: "Ver membros", Href: "/dashboard/members"},
			})
		}
	}

	if s.RecurringCount > 0 && s.Income > 0 {
		// A recorrencia ativa nao e a receita do periodo; usamos a contagem como
		// sinal de previsibilidade, sem inventar percentual sobre a arrecadacao.
		out = append(out, Insight{
			ID: "fin_recurring", Severity: Info, Title: "Doacoes recorrentes ativas",
			Message: fmt.Sprintf("%d agendamento(s) recorrente(s) somando %s por ciclo, o que ajuda a prever o caixa.", s.RecurringCount, brl(s.RecurringAmount)),
			Metric:  fmt.Sprintf("%d agendamentos", s.RecurringCount),
			Action:  &Action{Label: "Ver recorrencias", Href: "/dashboard/finance"},
		})
	}

	if s.LapsedGivers > 0 {
		out = append(out, Insight{
			ID: "fin_lapsed", Severity: Warn, Title: "Doadores que pararam",
			Message: fmt.Sprintf("%d pessoa(s) contribuiram nos ultimos 6 meses e nao contribuem ha mais de 1 mes.", s.LapsedGivers),
			Metric:  fmt.Sprintf("%d pessoas", s.LapsedGivers),
			Action:  &Action{Label: "Ver queda de contribuicao", Href: "/dashboard/reports/contribution-drop"},
		})
	}

	return out
}

func (r *Repo) peopleInsights(s Summary) []Insight {
	out := []Insight{}
	if s.Visitors > 0 {
		conv := (float64(s.ConvertedVisitors) / float64(s.Visitors)) * 100
		switch {
		case conv < visitorConvOK:
			out = append(out, Insight{
				ID: "peo_visitor_conv", Severity: Warn, Title: "Visitantes nao estao retornando",
				Message: fmt.Sprintf("Apenas %.0f%% dos %d visitantes do periodo viraram membro. Um follow-up em ate 7 dias costuma elevar esse numero.", conv, s.Visitors),
				Metric:  fmt.Sprintf("%.0f%%", conv), Benchmark: "acima de 20%",
				Action: &Action{Label: "Ver visitantes", Href: "/dashboard/visitors"},
			})
		case conv >= visitorConvGood:
			out = append(out, Insight{
				ID: "peo_visitor_conv", Severity: Good, Title: "Boa conversao de visitantes",
				Message: fmt.Sprintf("%.0f%% dos visitantes do periodo foram integrados como membros.", conv),
				Metric:  fmt.Sprintf("%.0f%%", conv), Benchmark: "acima de 40%",
			})
		}
	}

	if s.ActiveMembers > 0 {
		part := (float64(s.GroupMembers) / float64(s.ActiveMembers)) * 100
		if part < groupPartMin {
			out = append(out, Insight{
				ID: "peo_groups", Severity: Warn, Title: "Baixa participacao em grupos",
				Message: fmt.Sprintf("Apenas %.0f%% dos membros ativos estao em algum grupo/celula. Grupo e o principal sinal de conexao.", part),
				Metric:  fmt.Sprintf("%.0f%%", part), Benchmark: "acima de 40%",
				Action: &Action{Label: "Ver grupos", Href: "/dashboard/ministries"},
			})
		}
	}

	if s.TotalMembers > 0 {
		gap := (float64(s.MissingContact) / float64(s.TotalMembers)) * 100
		if gap > contactGapMax {
			out = append(out, Insight{
				ID: "peo_contact", Severity: Warn, Title: "Cadastros sem contato",
				Message: fmt.Sprintf("%.0f%% dos membros nao tem e-mail nem telefone, o que impede avisos e follow-up.", gap),
				Metric:  fmt.Sprintf("%.0f%%", gap), Benchmark: "abaixo de 10%",
				Action: &Action{Label: "Ver membros", Href: "/dashboard/members"},
			})
		}
	}
	return out
}

func (r *Repo) operationsInsights(s Summary) []Insight {
	out := []Insight{}
	if s.Events > 0 {
		avg := float64(s.Participants) / float64(s.Events)
		out = append(out, Insight{
			ID: "ope_attendance", Severity: Info, Title: "Media de participantes por evento",
			Message: fmt.Sprintf("%d evento(s) no periodo somaram %d participantes (%.1f por evento).", s.Events, s.Participants, avg),
			Metric:  fmt.Sprintf("%.1f por evento", avg),
			Action:  &Action{Label: "Ver participantes", Href: "/dashboard/reports/attendance"},
		})
	}
	if s.ZeroAttendance > 0 {
		out = append(out, Insight{
			ID: "ope_zero", Severity: Warn, Title: "Eventos sem chamada registrada",
			Message: fmt.Sprintf("%d evento(s) ja aconteceram sem nenhum participante lancado. Sem isso, os relatorios de presenca ficam incompletos.", s.ZeroAttendance),
			Metric:  fmt.Sprintf("%d eventos", s.ZeroAttendance),
			Action:  &Action{Label: "Abrir agenda", Href: "/dashboard/events"},
		})
	}
	if s.UpcomingNoRoster > 0 {
		out = append(out, Insight{
			ID: "ope_roster", Severity: Warn, Title: "Eventos sem escala nos proximos 30 dias",
			Message: fmt.Sprintf("%d evento(s) futuros ainda nao tem escala vinculada.", s.UpcomingNoRoster),
			Metric:  fmt.Sprintf("%d eventos", s.UpcomingNoRoster),
			Action:  &Action{Label: "Ver escalas", Href: "/dashboard/rosters"},
		})
	}
	if s.PendingAssignment > 0 {
		out = append(out, Insight{
			ID: "ope_pending", Severity: Info, Title: "Convites de escala aguardando resposta",
			Message: fmt.Sprintf("%d convite(s) de escala nos proximos 30 dias ainda estao como 'convidado'.", s.PendingAssignment),
			Metric:  fmt.Sprintf("%d convites", s.PendingAssignment),
			Action:  &Action{Label: "Ver escalas", Href: "/dashboard/rosters"},
		})
	}
	return out
}

func brl(v float64) string {
	return fmt.Sprintf("R$ %.2f", v)
}
