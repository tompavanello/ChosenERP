// Package programacao gerencia a GRADE de horarios recorrentes da igreja
// (cultos, relogio de oracao, celulas, EBD, ensaios, reunioes) e publica as
// ocorrencias na agenda de eventos (`church_events`).
//
// O tipo e FIXO (`kind`): o conjunto fechado evita que cada modulo crie o seu
// proprio rotulo. A ligacao com a agenda e feita por `church_events.origin`
// ('programacao') + `origin_id` (o id da programacao). O isolamento e do RLS.
package programacao

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Kinds e o conjunto FECHADO de tipos de programacao (espelha o CHECK de 000069
// e o catalogo selado event_kinds).
var Kinds = map[string]bool{
	"culto": true, "oracao": true, "celula": true, "ebd": true,
	"ensaio": true, "reuniao": true, "outro": true,
}

// Programacao e um horario fixo recorrente.
type Programacao struct {
	ID              string    `json:"id"`
	BranchID        string    `json:"branch_id"`
	Name            string    `json:"name"`
	Kind            string    `json:"kind"`
	KindName        *string   `json:"kind_name,omitempty"`
	KindColor       *string   `json:"kind_color,omitempty"`
	// Title e o titulo amigavel (usado quando kind='outro').
	Title           *string   `json:"title,omitempty"`
	Weekday         int       `json:"weekday"`
	StartTime       string    `json:"start_time"` // "HH:MM"
	DurationMinutes int       `json:"duration_minutes"`
	Location        *string   `json:"location,omitempty"`
	Notes           *string   `json:"notes,omitempty"`
	IsActive        bool      `json:"is_active"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateInput struct {
	Name            string  `json:"name"`
	Kind            string  `json:"kind"`
	Title           *string `json:"title"`
	Weekday         int     `json:"weekday"`
	StartTime       string  `json:"start_time"`
	DurationMinutes int     `json:"duration_minutes"`
	Location        *string `json:"location"`
	Notes           *string `json:"notes"`
	IsActive        *bool   `json:"is_active"`
	SortOrder       *int    `json:"sort_order"`
}

type UpdateInput struct {
	Name            *string `json:"name"`
	Kind            *string `json:"kind"`
	Title           *string `json:"title"`
	Weekday         *int    `json:"weekday"`
	StartTime       *string `json:"start_time"`
	DurationMinutes *int    `json:"duration_minutes"`
	Location        *string `json:"location"`
	Notes           *string `json:"notes"`
	IsActive        *bool   `json:"is_active"`
	SortOrder       *int    `json:"sort_order"`
}

type Repo struct{}

// cols faz o join de exibicao com o catalogo selado (nome/cor do tipo).
const cols = `p.id::text, p.branch_id::text, p.name, p.kind, k.name, k.color, p.title,
	p.weekday, to_char(p.start_time,'HH24:MI'), p.duration_minutes,
	p.location, p.notes, p.is_active, p.sort_order, p.created_at`

func scan(row pgx.Row) (*Programacao, error) {
	var p Programacao
	err := row.Scan(&p.ID, &p.BranchID, &p.Name, &p.Kind, &p.KindName, &p.KindColor, &p.Title,
		&p.Weekday, &p.StartTime, &p.DurationMinutes, &p.Location, &p.Notes,
		&p.IsActive, &p.SortOrder, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List devolve a grade do escopo (RLS), ordenada por tipo/dia/hora.
func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]Programacao, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+cols+`
		FROM programacoes p
		LEFT JOIN event_kinds k ON k.tenant_id = p.tenant_id AND k.slug = p.kind
		ORDER BY p.kind, p.weekday, p.start_time, p.sort_order, p.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Programacao{}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Programacao, error) {
	return scan(tx.QueryRow(ctx, `
		SELECT `+cols+`
		FROM programacoes p
		LEFT JOIN event_kinds k ON k.tenant_id = p.tenant_id AND k.slug = p.kind
		WHERE p.id = $1::uuid`, id))
}

func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, in CreateInput) (*Programacao, error) {
	dur := in.DurationMinutes
	if dur <= 0 {
		dur = 90
	}
	order := 0
	if in.SortOrder != nil {
		order = *in.SortOrder
	}
	kind := in.Kind
	if !Kinds[kind] {
		kind = "outro"
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO programacoes
			(tenant_id, branch_id, name, kind, title, weekday, start_time,
			 duration_minutes, location, notes, is_active, sort_order, created_by)
		VALUES ($1, $2, $3, $4, NULLIF(btrim($5),''), $6, $7::time, $8, $9, $10,
			COALESCE($11, true), $12, $13::uuid)
		RETURNING id::text`,
		tenantID, branchID, strings.TrimSpace(in.Name), kind, str(in.Title), in.Weekday,
		in.StartTime, dur, in.Location, in.Notes, in.IsActive, order, actorID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, id)
}

// Update edita a grade (PATCH).
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*Programacao, error) {
	var kindPtr interface{}
	if in.Kind != nil && Kinds[*in.Kind] {
		kindPtr = *in.Kind
	}
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE programacoes p SET
			name = COALESCE(NULLIF(btrim($2),''), p.name),
			kind = COALESCE($3, p.kind),
			title = CASE WHEN $11::boolean THEN NULLIF(btrim($12),'') ELSE p.title END,
			weekday = COALESCE($4, p.weekday),
			start_time = COALESCE($5::time, p.start_time),
			duration_minutes = COALESCE($6, p.duration_minutes),
			location = COALESCE($7, p.location),
			notes = COALESCE($8, p.notes),
			is_active = COALESCE($9, p.is_active),
			sort_order = COALESCE($10, p.sort_order),
			updated_at = now()
		WHERE p.id = $1::uuid
		RETURNING p.id::text`,
		id, in.Name, kindPtr, in.Weekday, in.StartTime, in.DurationMinutes,
		in.Location, in.Notes, in.IsActive, in.SortOrder,
		in.Title != nil, str(in.Title)).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, updatedID)
}

// Delete remove a definicao. As ocorrencias ja publicadas permanecem na agenda
// (`origin_id` fica orfao, sem FK); a grade futura nao e mais gerada.
func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM programacoes WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// GenerateEvents publica na agenda (church_events) as ocorrencias no periodo
// [from,to]. Sem id, publica TODAS as programacoes ativas que o usuario pode
// escrever. A hora local e convertida pelo timezone do tenant e a ocorrencia e
// marcada com origin='programacao'. As ocorrencias do periodo ja existentes sao
// ATUALIZADAS (titulo/tipo/notas) e as novas sao inseridas - idempotente, mas
// reflete a grade atual. Devolve quantas foram criadas e quantas atualizadas.
func (r *Repo) GenerateEvents(ctx context.Context, tx pgx.Tx, programacaoID, from, to, actorID string) (created, updated int64, err error) {
	// 1) Reflete a grade atual nas ocorrencias ja publicadas (ex.: o titulo de
	//    um evento de tipo 'outro' foi definido depois de ja publicado antes).
	tag, err := tx.Exec(ctx, `
		UPDATE church_events e SET
			title = p.title,
			kind_id = k.id,
			notes = p.notes,
			updated_at = now()
		FROM programacoes p
		LEFT JOIN event_kinds k ON k.tenant_id = p.tenant_id AND k.slug = p.kind
		WHERE e.origin = 'programacao' AND e.origin_id = p.id
		  AND (NULLIF($1,'')::uuid IS NULL OR p.id = NULLIF($1,'')::uuid)
		  AND (e.title IS DISTINCT FROM p.title
		       OR e.kind_id IS DISTINCT FROM k.id
		       OR e.notes IS DISTINCT FROM p.notes)`,
		programacaoID)
	if err != nil {
		return 0, 0, err
	}
	updated = tag.RowsAffected()

	// 2) Insere as ocorrencias novas do periodo.
	tag2, err := tx.Exec(ctx, `
		INSERT INTO church_events
			(tenant_id, branch_id, kind_id, title, starts_at, ends_at, attendance_mode, notes, created_by, origin, origin_id)
		SELECT p.tenant_id, p.branch_id, k.id, p.title,
		       ((g.d::date + p.start_time) AT TIME ZONE tz.tz),
		       ((g.d::date + p.start_time + make_interval(mins => p.duration_minutes)) AT TIME ZONE tz.tz),
		       'nominal', p.notes, $4::uuid, 'programacao', p.id
		FROM programacoes p
		LEFT JOIN event_kinds k ON k.tenant_id = p.tenant_id AND k.slug = p.kind
		CROSS JOIN LATERAL (
			SELECT COALESCE(NULLIF(t.timezone,''),'America/Sao_Paulo') AS tz
			FROM tenants t WHERE t.id = p.tenant_id
		) tz
		CROSS JOIN generate_series($2::date, $3::date, interval '1 day') AS g(d)
		WHERE p.is_active
		  AND rls_write(p.tenant_id, p.branch_id, false)
		  AND (NULLIF($1,'')::uuid IS NULL OR p.id = NULLIF($1,'')::uuid)
		  AND EXTRACT(DOW FROM g.d)::int = p.weekday
		  AND NOT EXISTS (
		      SELECT 1 FROM church_events e
		      WHERE e.origin = 'programacao' AND e.origin_id = p.id
		        AND e.starts_at = ((g.d::date + p.start_time) AT TIME ZONE tz.tz)
		  )`,
		programacaoID, from, to, actorID)
	if err != nil {
		return 0, updated, err
	}
	created = tag2.RowsAffected()
	return created, updated, nil
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
