package events

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- Relatorio de participantes por evento ----
//
// Compara o numero de participantes dos eventos do periodo escolhido com o mesmo
// intervalo do ano anterior. Alem do resumo geral, entrega a quebra por tipo de
// evento, a evolucao mensal, o ranking dos eventos com mais participantes e a
// lista completa de eventos do periodo.

type ReportPeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type AttendanceSummary struct {
	Events               int     `json:"events"`
	Participants         int     `json:"participants"`
	AvgPerEvent          float64 `json:"avg_per_event"`
	Invited              int     `json:"invited"`
	PrevEvents           int     `json:"prev_events"`
	PrevParticipants     int     `json:"prev_participants"`
	PrevAvgPerEvent      float64 `json:"prev_avg_per_event"`
	DeltaParticipantsPct float64 `json:"delta_participants_pct"`
	DeltaEventsPct       float64 `json:"delta_events_pct"`
	DeltaAvgPct          float64 `json:"delta_avg_pct"`
}

type AttendanceKindRow struct {
	KindID           string  `json:"kind_id"`
	KindName         string  `json:"kind_name"`
	Events           int     `json:"events"`
	Participants     int     `json:"participants"`
	AvgPerEvent      float64 `json:"avg_per_event"`
	PrevEvents       int     `json:"prev_events"`
	PrevParticipants int     `json:"prev_participants"`
	DeltaPct         float64 `json:"delta_pct"`
}

type AttendanceEventRow struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	KindID          string    `json:"kind_id"`
	KindName        string    `json:"kind_name"`
	BranchID        string    `json:"branch_id"`
	BranchName      string    `json:"branch_name"`
	StartsAt        time.Time `json:"starts_at"`
	Participants    int       `json:"participants"`
	AttendanceCount int       `json:"attendance_count"`
	AttendanceMode  string    `json:"attendance_mode"`
	InvitedCount    int       `json:"invited_count"`
}

type AttendanceMonthRow struct {
	Month            string `json:"month"`
	Events           int    `json:"events"`
	Participants     int    `json:"participants"`
	PrevEvents       int    `json:"prev_events"`
	PrevParticipants int    `json:"prev_participants"`
}

type AttendanceReport struct {
	Period     ReportPeriod         `json:"period"`
	PrevPeriod ReportPeriod         `json:"prev_period"`
	Summary    AttendanceSummary    `json:"summary"`
	ByKind     []AttendanceKindRow  `json:"by_kind"`
	Monthly    []AttendanceMonthRow `json:"monthly"`
	TopEvents  []AttendanceEventRow `json:"top_events"`
	Events     []AttendanceEventRow `json:"events"`
}

type attendanceAgg struct {
	Events       int
	Participants int
	Invited      int
}

// BuildAttendanceReport monta o relatorio no escopo do RLS da transacao. `from`
// e `to` vazios assumem o ano corrente (1o de janeiro ate hoje). `kindID` vazio
// considera todos os tipos de evento.
func (r *Repo) BuildAttendanceReport(ctx context.Context, tx pgx.Tx, from, to, kindID string) (AttendanceReport, error) {
	from, to = resolveAttendancePeriod(from, to)
	rep := AttendanceReport{
		Period:    ReportPeriod{From: from, To: to},
		ByKind:    []AttendanceKindRow{},
		Monthly:   []AttendanceMonthRow{},
		TopEvents: []AttendanceEventRow{},
		Events:    []AttendanceEventRow{},
	}

	// Mesmo intervalo, um ano atras.
	if err := tx.QueryRow(ctx, `
		SELECT ($1::date - interval '1 year')::date::text,
		       ($2::date - interval '1 year')::date::text`, from, to).
		Scan(&rep.PrevPeriod.From, &rep.PrevPeriod.To); err != nil {
		return rep, err
	}

	events, err := listAttendanceEvents(ctx, tx, from, to, kindID)
	if err != nil {
		return rep, err
	}
	rep.Events = events

	cur, err := sumAttendance(ctx, tx, from, to, kindID)
	if err != nil {
		return rep, err
	}
	prev, err := sumAttendance(ctx, tx, rep.PrevPeriod.From, rep.PrevPeriod.To, kindID)
	if err != nil {
		return rep, err
	}
	rep.Summary = AttendanceSummary{
		Events:               cur.Events,
		Participants:         cur.Participants,
		Invited:              cur.Invited,
		PrevEvents:           prev.Events,
		PrevParticipants:     prev.Participants,
		AvgPerEvent:          avg(cur.Participants, cur.Events),
		PrevAvgPerEvent:      avg(prev.Participants, prev.Events),
		DeltaParticipantsPct: deltaPct(cur.Participants, prev.Participants),
		DeltaEventsPct:       deltaPct(cur.Events, prev.Events),
		DeltaAvgPct:          deltaPctF(avg(cur.Participants, cur.Events), avg(prev.Participants, prev.Events)),
	}

	// Ranking dos eventos com mais participantes.
	top := make([]AttendanceEventRow, len(events))
	copy(top, events)
	sort.SliceStable(top, func(i, j int) bool {
		if top[i].Participants == top[j].Participants {
			return top[i].StartsAt.Before(top[j].StartsAt)
		}
		return top[i].Participants > top[j].Participants
	})
	if len(top) > 10 {
		top = top[:10]
	}
	rep.TopEvents = top

	rep.ByKind, err = attendanceByKind(ctx, tx, from, to, rep.PrevPeriod.From, rep.PrevPeriod.To, kindID)
	if err != nil {
		return rep, err
	}
	rep.Monthly, err = attendanceMonthly(ctx, tx, from, to, rep.PrevPeriod.From, rep.PrevPeriod.To, kindID)
	if err != nil {
		return rep, err
	}
	return rep, nil
}

// resolveAttendancePeriod preenche from/to vazios com o ano corrente.
func resolveAttendancePeriod(from, to string) (string, string) {
	now := time.Now()
	if from == "" {
		from = fmt.Sprintf("%d-01-01", now.Year())
	}
	if to == "" {
		to = now.Format("2006-01-02")
	}
	return from, to
}

func sumAttendance(ctx context.Context, tx pgx.Tx, from, to, kindID string) (attendanceAgg, error) {
	var a attendanceAgg
	err := tx.QueryRow(ctx, `
		SELECT count(*)::int,
		       COALESCE(sum(e.participants_count), 0)::int,
		       COALESCE(sum((SELECT count(*) FROM event_invitees i WHERE i.event_id = e.id)), 0)::int
		FROM church_events e
		WHERE e.starts_at::date >= $1::date
		  AND e.starts_at::date <= $2::date
		  AND (NULLIF($3, '')::uuid IS NULL OR e.kind_id = NULLIF($3, '')::uuid)`,
		from, to, kindID).Scan(&a.Events, &a.Participants, &a.Invited)
	return a, err
}

func listAttendanceEvents(ctx context.Context, tx pgx.Tx, from, to, kindID string) ([]AttendanceEventRow, error) {
	rows, err := tx.Query(ctx, `
		SELECT e.id::text,
		       COALESCE(NULLIF(e.title, ''), k.name, 'Evento'),
		       COALESCE(e.kind_id::text, ''),
		       COALESCE(k.name, 'Sem tipo'),
		       e.branch_id::text,
		       COALESCE(b.name, ''),
		       e.starts_at,
		       e.participants_count,
		       (SELECT count(*) FROM event_attendance a WHERE a.event_id = e.id AND a.present)::int,
		       e.attendance_mode,
		       (SELECT count(*) FROM event_invitees i WHERE i.event_id = e.id)::int
		FROM church_events e
		LEFT JOIN event_kinds k ON k.id = e.kind_id
		LEFT JOIN branches b ON b.id = e.branch_id
		WHERE e.starts_at::date >= $1::date
		  AND e.starts_at::date <= $2::date
		  AND (NULLIF($3, '')::uuid IS NULL OR e.kind_id = NULLIF($3, '')::uuid)
		ORDER BY e.starts_at DESC`, from, to, kindID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AttendanceEventRow{}
	for rows.Next() {
		var e AttendanceEventRow
		if err := rows.Scan(&e.ID, &e.Title, &e.KindID, &e.KindName, &e.BranchID, &e.BranchName,
			&e.StartsAt, &e.Participants, &e.AttendanceCount, &e.AttendanceMode, &e.InvitedCount); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// attendanceByKind agrega participantes por tipo de evento nos dois periodos e
// calcula a variacao de cada tipo.
func attendanceByKind(ctx context.Context, tx pgx.Tx, from, to, prevFrom, prevTo, kindID string) ([]AttendanceKindRow, error) {
	cur, err := attendanceByKindPeriod(ctx, tx, from, to, kindID)
	if err != nil {
		return nil, err
	}
	prev, err := attendanceByKindPeriod(ctx, tx, prevFrom, prevTo, kindID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := []AttendanceKindRow{}
	for id, c := range cur {
		seen[id] = true
		p := prev[id]
		row := AttendanceKindRow{
			KindID:           id,
			KindName:         c.Name,
			Events:           c.Events,
			Participants:     c.Participants,
			PrevEvents:       p.Events,
			PrevParticipants: p.Participants,
			AvgPerEvent:      avg(c.Participants, c.Events),
		}
		row.DeltaPct = deltaPct(c.Participants, p.Participants)
		out = append(out, row)
	}
	for id, p := range prev {
		if seen[id] {
			continue
		}
		out = append(out, AttendanceKindRow{
			KindID:           id,
			KindName:         p.Name,
			PrevEvents:       p.Events,
			PrevParticipants: p.Participants,
			DeltaPct:         deltaPct(0, p.Participants),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Participants == out[j].Participants {
			return out[i].KindName < out[j].KindName
		}
		return out[i].Participants > out[j].Participants
	})
	return out, nil
}

type kindAgg struct {
	Name         string
	Events       int
	Participants int
}

func attendanceByKindPeriod(ctx context.Context, tx pgx.Tx, from, to, kindID string) (map[string]kindAgg, error) {
	rows, err := tx.Query(ctx, `
		SELECT COALESCE(e.kind_id::text, ''),
		       COALESCE(k.name, 'Sem tipo'),
		       count(*)::int,
		       COALESCE(sum(e.participants_count), 0)::int
		FROM church_events e
		LEFT JOIN event_kinds k ON k.id = e.kind_id
		WHERE e.starts_at::date >= $1::date
		  AND e.starts_at::date <= $2::date
		  AND (NULLIF($3, '')::uuid IS NULL OR e.kind_id = NULLIF($3, '')::uuid)
		GROUP BY 1, 2`, from, to, kindID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]kindAgg{}
	for rows.Next() {
		var id string
		var a kindAgg
		if err := rows.Scan(&id, &a.Name, &a.Events, &a.Participants); err != nil {
			return nil, err
		}
		out[id] = a
	}
	return out, rows.Err()
}

type monthAgg struct {
	Events       int
	Participants int
}

// attendanceMonthly devolve a serie mensal dos dois periodos alinhados pelo mes
// do calendario (ex.: marco/2026 vs marco/2025).
func attendanceMonthly(ctx context.Context, tx pgx.Tx, from, to, prevFrom, prevTo, kindID string) ([]AttendanceMonthRow, error) {
	cur, err := attendanceMonthlyPeriod(ctx, tx, from, to, kindID)
	if err != nil {
		return nil, err
	}
	prev, err := attendanceMonthlyPeriod(ctx, tx, prevFrom, prevTo, kindID)
	if err != nil {
		return nil, err
	}
	// Projeta os meses do ano anterior sobre o ano corrente.
	prevNorm := map[string]monthAgg{}
	for m, a := range prev {
		t, err := time.Parse("2006-01", m)
		if err != nil {
			continue
		}
		prevNorm[t.AddDate(1, 0, 0).Format("2006-01")] = a
	}
	seen := map[string]bool{}
	out := []AttendanceMonthRow{}
	for m, c := range cur {
		seen[m] = true
		p := prevNorm[m]
		out = append(out, AttendanceMonthRow{
			Month: m, Events: c.Events, Participants: c.Participants,
			PrevEvents: p.Events, PrevParticipants: p.Participants,
		})
	}
	for m, p := range prevNorm {
		if seen[m] {
			continue
		}
		out = append(out, AttendanceMonthRow{
			Month: m, PrevEvents: p.Events, PrevParticipants: p.Participants,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Month < out[j].Month })
	return out, nil
}

func attendanceMonthlyPeriod(ctx context.Context, tx pgx.Tx, from, to, kindID string) (map[string]monthAgg, error) {
	rows, err := tx.Query(ctx, `
		SELECT to_char(date_trunc('month', e.starts_at), 'YYYY-MM'),
		       count(*)::int,
		       COALESCE(sum(e.participants_count), 0)::int
		FROM church_events e
		WHERE e.starts_at::date >= $1::date
		  AND e.starts_at::date <= $2::date
		  AND (NULLIF($3, '')::uuid IS NULL OR e.kind_id = NULLIF($3, '')::uuid)
		GROUP BY 1
		ORDER BY 1`, from, to, kindID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]monthAgg{}
	for rows.Next() {
		var m string
		var a monthAgg
		if err := rows.Scan(&m, &a.Events, &a.Participants); err != nil {
			return nil, err
		}
		out[m] = a
	}
	return out, rows.Err()
}

// ---- helpers ----

func avg(total, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(total) / float64(n)
}

func deltaPct(cur, prev int) float64 {
	if prev == 0 {
		if cur == 0 {
			return 0
		}
		return 100
	}
	return (float64(cur-prev) / float64(prev)) * 100
}

func deltaPctF(cur, prev float64) float64 {
	if prev == 0 {
		if cur == 0 {
			return 0
		}
		return 100
	}
	return ((cur - prev) / prev) * 100
}
