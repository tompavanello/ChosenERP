// Package cultos gerencia a GRADE de horarios recorrentes dos cultos da igreja
// e publica as ocorrencias na agenda de eventos (`church_events`). O isolamento
// e do RLS; a ligacao com a agenda e feita por `church_events.culto_id`.
package cultos

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Culto e um horario fixo de culto (dia da semana + hora + duracao).
type Culto struct {
	ID              string    `json:"id"`
	BranchID        string    `json:"branch_id"`
	Name            string    `json:"name"`
	EventKindID     *string   `json:"event_kind_id,omitempty"`
	EventKindName   *string   `json:"event_kind_name,omitempty"`
	EventKindColor  *string   `json:"event_kind_color,omitempty"`
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
	EventKindID     *string `json:"event_kind_id"`
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
	EventKindID     *string `json:"event_kind_id"`
	Weekday         *int    `json:"weekday"`
	StartTime       *string `json:"start_time"`
	DurationMinutes *int    `json:"duration_minutes"`
	Location        *string `json:"location"`
	Notes           *string `json:"notes"`
	IsActive        *bool   `json:"is_active"`
	SortOrder       *int    `json:"sort_order"`
}

type Repo struct{}

const cols = `c.id::text, c.branch_id::text, c.name, c.event_kind_id::text,
	k.name, k.color, c.weekday, to_char(c.start_time,'HH24:MI'),
	c.duration_minutes, c.location, c.notes, c.is_active, c.sort_order, c.created_at`

func scan(row pgx.Row) (*Culto, error) {
	var c Culto
	err := row.Scan(&c.ID, &c.BranchID, &c.Name, &c.EventKindID, &c.EventKindName,
		&c.EventKindColor, &c.Weekday, &c.StartTime, &c.DurationMinutes,
		&c.Location, &c.Notes, &c.IsActive, &c.SortOrder, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// List devolve a grade do escopo (RLS), ordenada por dia/hora.
func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]Culto, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+cols+`
		FROM cultos c
		LEFT JOIN event_kinds k ON k.id = c.event_kind_id
		ORDER BY c.weekday, c.start_time, c.sort_order, c.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Culto{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Culto, error) {
	return scan(tx.QueryRow(ctx, `
		SELECT `+cols+`
		FROM cultos c
		LEFT JOIN event_kinds k ON k.id = c.event_kind_id
		WHERE c.id = $1::uuid`, id))
}

func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, in CreateInput) (*Culto, error) {
	dur := in.DurationMinutes
	if dur <= 0 {
		dur = 90
	}
	order := 0
	if in.SortOrder != nil {
		order = *in.SortOrder
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO cultos
			(tenant_id, branch_id, name, event_kind_id, weekday, start_time,
			 duration_minutes, location, notes, is_active, sort_order, created_by)
		VALUES ($1, $2, $3, NULLIF($4,'')::uuid, $5, $6::time, $7, $8, $9,
			COALESCE($10, true), $11, $12::uuid)
		RETURNING id::text`,
		tenantID, branchID, strings.TrimSpace(in.Name), str(in.EventKindID),
		in.Weekday, in.StartTime, dur, in.Location, in.Notes, in.IsActive, order, actorID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, id)
}

// Update edita a grade (PATCH). `event_kind_id` vazio LIMPA o tipo (NULL).
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*Culto, error) {
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE cultos c SET
			name = COALESCE(NULLIF(btrim($2),''), c.name),
			event_kind_id = CASE WHEN $3::boolean THEN NULLIF($4,'')::uuid ELSE c.event_kind_id END,
			weekday = COALESCE($5, c.weekday),
			start_time = COALESCE($6::time, c.start_time),
			duration_minutes = COALESCE($7, c.duration_minutes),
			location = COALESCE($8, c.location),
			notes = COALESCE($9, c.notes),
			is_active = COALESCE($10, c.is_active),
			sort_order = COALESCE($11, c.sort_order),
			updated_at = now()
		WHERE c.id = $1::uuid
		RETURNING c.id::text`,
		id, in.Name, in.EventKindID != nil, str(in.EventKindID), in.Weekday,
		in.StartTime, in.DurationMinutes, in.Location, in.Notes, in.IsActive,
		in.SortOrder).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, updatedID)
}

// Delete remove a definicao do culto. Os eventos ja publicados permanecem na
// agenda (o FK e ON DELETE SET NULL); a grade futura nao e mais gerada.
func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM cultos WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// GenerateEvents publica na agenda (church_events) as ocorrencias dos cultos no
// periodo [from,to] (datas "YYYY-MM-DD"). Sem cultoID, publica TODOS os cultos
// ativos que o usuario pode escrever. A hora local e convertida com o timezone
// do tenant (`AT TIME ZONE`), entao o horario fica correto no calendario.
//
// E idempotente: pula ocorrencias que ja existem para o mesmo culto + horario,
// de forma que publicar o mesmo mes duas vezes nao duplica nada. Devolve quantos
// eventos foram criados.
func (r *Repo) GenerateEvents(ctx context.Context, tx pgx.Tx, cultoID, from, to, actorID string) (int64, error) {
	tag, err := tx.Exec(ctx, `
		INSERT INTO church_events
			(tenant_id, branch_id, kind_id, starts_at, ends_at, attendance_mode, notes, created_by, culto_id)
		SELECT c.tenant_id, c.branch_id, c.event_kind_id,
		       ((g.d::date + c.start_time) AT TIME ZONE tz.tz),
		       ((g.d::date + c.start_time + make_interval(mins => c.duration_minutes)) AT TIME ZONE tz.tz),
		       'nominal', c.notes, $4::uuid, c.id
		FROM cultos c
		CROSS JOIN LATERAL (
			SELECT COALESCE(NULLIF(t.timezone,''),'America/Sao_Paulo') AS tz
			FROM tenants t WHERE t.id = c.tenant_id
		) tz
		CROSS JOIN generate_series($2::date, $3::date, interval '1 day') AS g(d)
		WHERE c.is_active
		  AND rls_write(c.tenant_id, c.branch_id, false)
		  AND (NULLIF($1,'')::uuid IS NULL OR c.id = NULLIF($1,'')::uuid)
		  AND EXTRACT(DOW FROM g.d)::int = c.weekday
		  AND NOT EXISTS (
		      SELECT 1 FROM church_events e
		      WHERE e.culto_id = c.id
		        AND e.starts_at = ((g.d::date + c.start_time) AT TIME ZONE tz.tz)
		  )`,
		cultoID, from, to, actorID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
