// Package events gerencia o registro de eventos da igreja, a chamada nominal de
// presenca e o historico de frequencia dos membros. O isolamento e do RLS.
package events

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- Tipos de evento ----

type Kind struct {
	ID        string  `json:"id"`
	BranchID  *string `json:"branch_id,omitempty"`
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	SortOrder int     `json:"sort_order"`
	IsActive  bool    `json:"is_active"`
	Color     *string `json:"color,omitempty"`
}

type KindInput struct {
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	SortOrder int     `json:"sort_order"`
	IsActive  *bool   `json:"is_active"`
	Color     *string `json:"color"`
}

// ---- Eventos ----

type Event struct {
	ID                string     `json:"id"`
	BranchID          string     `json:"branch_id"`
	KindID            *string    `json:"kind_id,omitempty"`
	KindName          *string    `json:"kind_name,omitempty"`
	StartsAt          time.Time  `json:"starts_at"`
	EndsAt            *time.Time `json:"ends_at,omitempty"`
	ParticipantsCount int        `json:"participants_count"`
	AttendanceCount   int        `json:"attendance_count"`
	// AttendanceMode: 'nominal' (chamada) ou 'count' (so o numero).
	AttendanceMode string    `json:"attendance_mode"`
	EstimatedCost  *float64  `json:"estimated_cost,omitempty"`
	CostActual     float64   `json:"cost_actual"`
	InvitedCount   int       `json:"invited_count"`
	Notes          *string   `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateInput struct {
	KindID            *string  `json:"kind_id"`
	StartsAt          string   `json:"starts_at"`
	EndsAt            *string  `json:"ends_at"`
	ParticipantsCount int      `json:"participants_count"`
	AttendanceMode    *string  `json:"attendance_mode"`
	EstimatedCost     *float64 `json:"estimated_cost"`
	Notes             *string  `json:"notes"`
}

type UpdateInput struct {
	KindID            *string  `json:"kind_id"`
	StartsAt          *string  `json:"starts_at"`
	EndsAt            *string  `json:"ends_at"`
	ParticipantsCount *int     `json:"participants_count"`
	AttendanceMode    *string  `json:"attendance_mode"`
	EstimatedCost     *float64 `json:"estimated_cost"`
	Notes             *string  `json:"notes"`
}

// Invitee e uma pessoa ou ministerio convocado para o evento.
type Invitee struct {
	ID           string  `json:"id"`
	EventID      string  `json:"event_id"`
	MemberID     *string `json:"member_id,omitempty"`
	MemberName   *string `json:"member_name,omitempty"`
	MinistryID   *string `json:"ministry_id,omitempty"`
	MinistryName *string `json:"ministry_name,omitempty"`
}

type Attendance struct {
	ID         string `json:"id"`
	EventID    string `json:"event_id"`
	MemberID   string `json:"member_id"`
	MemberName string `json:"member_name"`
	Present    bool   `json:"present"`
}

// Frequency e uma entrada do historico de frequencia do membro.
type Frequency struct {
	ID        string  `json:"id"`
	MemberID  string  `json:"member_id"`
	Frequency string  `json:"frequency"`
	StartedAt string  `json:"started_at"`
	EndedAt   *string `json:"ended_at,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

type Repo struct{}

// ---- Tipos de evento ----

func (r *Repo) ListKinds(ctx context.Context, tx pgx.Tx) ([]Kind, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, branch_id::text, name, slug, sort_order, is_active, color
		FROM event_kinds ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Kind{}
	for rows.Next() {
		var k Kind
		var bid *string
		if err := rows.Scan(&k.ID, &bid, &k.Name, &k.Slug, &k.SortOrder, &k.IsActive, &k.Color); err != nil {
			return nil, err
		}
		k.BranchID = bid
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *Repo) CreateKind(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in KindInput) (*Kind, error) {
	var k Kind
	err := tx.QueryRow(ctx, `
		INSERT INTO event_kinds (tenant_id, branch_id, name, slug, sort_order, is_active, color)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5, COALESCE($6::boolean, true), $7)
		RETURNING id::text, branch_id::text, name, slug, sort_order, is_active, color`,
		tenantID, branchID, in.Name, in.Slug, in.SortOrder, in.IsActive, in.Color).
		Scan(&k.ID, &k.BranchID, &k.Name, &k.Slug, &k.SortOrder, &k.IsActive, &k.Color)
	return &k, err
}

func (r *Repo) UpdateKind(ctx context.Context, tx pgx.Tx, id string, in KindInput) (*Kind, error) {
	var k Kind
	err := tx.QueryRow(ctx, `
		UPDATE event_kinds SET
			name = COALESCE(NULLIF($2,''), name),
			slug = COALESCE(NULLIF($3,''), slug),
			sort_order = $4,
			is_active = COALESCE($5::boolean, is_active),
			color = COALESCE($6, color)
		WHERE id = $1::uuid
		RETURNING id::text, branch_id::text, name, slug, sort_order, is_active, color`,
		id, in.Name, in.Slug, in.SortOrder, in.IsActive, in.Color).
		Scan(&k.ID, &k.BranchID, &k.Name, &k.Slug, &k.SortOrder, &k.IsActive, &k.Color)
	return &k, err
}

// DeleteKind recusa excluir um tipo ainda usado por eventos (sugere desativar).
func (r *Repo) DeleteKind(ctx context.Context, tx pgx.Tx, id string) error {
	var used int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM church_events WHERE kind_id = $1::uuid`, id).Scan(&used); err != nil {
		return err
	}
	if used > 0 {
		return ErrKindInUse
	}
	tag, err := tx.Exec(ctx, `DELETE FROM event_kinds WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ---- Eventos ----

const eventCols = `e.id::text, e.branch_id::text, e.kind_id::text, k.name,
	e.starts_at, e.ends_at, e.participants_count,
	(SELECT count(*) FROM event_attendance a WHERE a.event_id = e.id AND a.present)::int,
	e.attendance_mode, e.estimated_cost::float8,
	(SELECT count(*) FROM event_invitees i WHERE i.event_id = e.id)::int,
	(SELECT COALESCE(SUM(a.amount),0) FROM financial_event_allocations a WHERE a.event_id = e.id)::float8,
	e.notes, e.created_at`

func scanEvent(row pgx.Row) (*Event, error) {
	var e Event
	err := row.Scan(&e.ID, &e.BranchID, &e.KindID, &e.KindName, &e.StartsAt, &e.EndsAt,
		&e.ParticipantsCount, &e.AttendanceCount, &e.AttendanceMode, &e.EstimatedCost,
		&e.InvitedCount, &e.CostActual, &e.Notes, &e.CreatedAt)
	return &e, err
}

func (r *Repo) ListEvents(ctx context.Context, tx pgx.Tx, from, to, kindID string) ([]Event, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+eventCols+`
		FROM church_events e
		LEFT JOIN event_kinds k ON k.id = e.kind_id
		WHERE ($1 = '' OR e.starts_at::date >= $1::date)
		  AND ($2 = '' OR e.starts_at::date <= $2::date)
		  AND ($3 = '' OR e.kind_id = $3::uuid)
		ORDER BY e.starts_at DESC`, from, to, kindID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (r *Repo) GetEvent(ctx context.Context, tx pgx.Tx, id string) (*Event, error) {
	return scanEvent(tx.QueryRow(ctx, `
		SELECT `+eventCols+`
		FROM church_events e
		LEFT JOIN event_kinds k ON k.id = e.kind_id
		WHERE e.id = $1::uuid`, id))
}

func (r *Repo) CreateEvent(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, in CreateInput) (*Event, error) {
	starts, err := parseTime(in.StartsAt)
	if err != nil {
		return nil, err
	}
	var ends *time.Time
	if in.EndsAt != nil && *in.EndsAt != "" {
		t, err := parseTime(*in.EndsAt)
		if err != nil {
			return nil, err
		}
		ends = &t
	}
	var newID string
	mode := "nominal"
	if in.AttendanceMode != nil && (*in.AttendanceMode == "nominal" || *in.AttendanceMode == "count") {
		mode = *in.AttendanceMode
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO church_events (tenant_id, branch_id, kind_id, starts_at, ends_at, participants_count, attendance_mode, estimated_cost, notes, created_by)
		VALUES ($1, NULLIF($2,'')::uuid, NULLIF($3,'')::uuid, $4, $5, $6, $7, $8, $9, $10::uuid)
		RETURNING id::text`,
		tenantID, branchID, str(in.KindID), starts, ends, in.ParticipantsCount, mode, in.EstimatedCost, in.Notes, actorID).Scan(&newID)
	if err != nil {
		return nil, err
	}
	return r.GetEvent(ctx, tx, newID)
}

func (r *Repo) UpdateEvent(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*Event, error) {
	var startsPtr, endsPtr interface{}
	if in.StartsAt != nil && *in.StartsAt != "" {
		t, err := parseTime(*in.StartsAt)
		if err != nil {
			return nil, err
		}
		startsPtr = t
	}
	if in.EndsAt != nil && *in.EndsAt != "" {
		t, err := parseTime(*in.EndsAt)
		if err != nil {
			return nil, err
		}
		endsPtr = t
	}
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE church_events e SET
			kind_id = CASE WHEN $2::boolean THEN NULLIF($3,'')::uuid ELSE e.kind_id END,
			starts_at = COALESCE($4::timestamptz, e.starts_at),
			ends_at = COALESCE($5::timestamptz, e.ends_at),
			participants_count = COALESCE($6, e.participants_count),
			notes = COALESCE($7, e.notes),
			attendance_mode = COALESCE($8, e.attendance_mode),
			estimated_cost = COALESCE($9, e.estimated_cost),
			updated_at = now()
		WHERE e.id = $1::uuid
		RETURNING e.id::text`,
		id, in.KindID != nil, str(in.KindID), startsPtr, endsPtr, in.ParticipantsCount, in.Notes,
		in.AttendanceMode, in.EstimatedCost).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	return r.GetEvent(ctx, tx, updatedID)
}

func (r *Repo) DeleteEvent(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM church_events WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ---- Convocados (pessoas / ministerios) ----

func (r *Repo) ListInvitees(ctx context.Context, tx pgx.Tx, eventID string) ([]Invitee, error) {
	rows, err := tx.Query(ctx, `
		SELECT i.id::text, i.event_id::text, i.member_id::text, m.full_name,
		       i.ministry_id::text, mi.name
		FROM event_invitees i
		LEFT JOIN members m ON m.id = i.member_id
		LEFT JOIN ministries mi ON mi.id = i.ministry_id
		WHERE i.event_id = $1::uuid
		ORDER BY mi.name NULLS LAST, m.full_name`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Invitee{}
	for rows.Next() {
		var i Invitee
		if err := rows.Scan(&i.ID, &i.EventID, &i.MemberID, &i.MemberName, &i.MinistryID, &i.MinistryName); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// SetInvitees substitui a lista de convocados do evento. So aceita pessoas e
// ministerios do mesmo tenant/filial do evento.
func (r *Repo) SetInvitees(ctx context.Context, tx pgx.Tx, eventID string, memberIDs, ministryIDs []string) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM church_events WHERE id = $1::uuid)`, eventID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	if _, err := tx.Exec(ctx, `DELETE FROM event_invitees WHERE event_id = $1::uuid`, eventID); err != nil {
		return err
	}
	for _, mid := range memberIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_invitees (tenant_id, branch_id, event_id, member_id)
			SELECT e.tenant_id, e.branch_id, e.id, m.id
			FROM church_events e
			JOIN members m ON m.id = $2::uuid AND m.tenant_id = e.tenant_id AND m.branch_id = e.branch_id
			WHERE e.id = $1::uuid`, eventID, mid); err != nil {
			return err
		}
	}
	for _, minID := range ministryIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_invitees (tenant_id, branch_id, event_id, ministry_id)
			SELECT e.tenant_id, e.branch_id, e.id, mi.id
			FROM church_events e
			JOIN ministries mi ON mi.id = $2::uuid AND mi.tenant_id = e.tenant_id AND mi.branch_id = e.branch_id
			WHERE e.id = $1::uuid`, eventID, minID); err != nil {
			return err
		}
	}
	return nil
}

// ---- Chamada nominal ----

func (r *Repo) ListAttendance(ctx context.Context, tx pgx.Tx, eventID string) ([]Attendance, error) {
	rows, err := tx.Query(ctx, `
		SELECT a.id::text, a.event_id::text, a.member_id::text, m.full_name, a.present
		FROM event_attendance a
		JOIN members m ON m.id = a.member_id
		WHERE a.event_id = $1::uuid
		ORDER BY m.full_name`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Attendance{}
	for rows.Next() {
		var a Attendance
		if err := rows.Scan(&a.ID, &a.EventID, &a.MemberID, &a.MemberName, &a.Present); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// SaveAttendance substitui a chamada do evento pela lista informada e ajusta o
// total de participantes. As duas informacoes coexistem: `total` (digitado) e a
// presenca nominal.
func (r *Repo) SaveAttendance(ctx context.Context, tx pgx.Tx, eventID string, total int, presentIDs []string) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM church_events WHERE id = $1::uuid)`, eventID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	if _, err := tx.Exec(ctx, `DELETE FROM event_attendance WHERE event_id = $1::uuid`, eventID); err != nil {
		return err
	}
	for _, mid := range presentIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_attendance (tenant_id, branch_id, event_id, member_id, present)
			SELECT e.tenant_id, e.branch_id, e.id, m.id, true
			FROM church_events e
			JOIN members m ON m.id = $2::uuid AND m.tenant_id = e.tenant_id AND m.branch_id = e.branch_id
			WHERE e.id = $1::uuid
			ON CONFLICT (event_id, member_id) DO UPDATE SET present = true`,
			eventID, mid); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `UPDATE church_events SET participants_count = $2, updated_at = now() WHERE id = $1::uuid`, eventID, total)
	return err
}

// ---- Frequencia do membro ----

func (r *Repo) ListFrequency(ctx context.Context, tx pgx.Tx, memberID string) ([]Frequency, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, member_id::text, frequency, started_at::text, ended_at::text, notes
		FROM member_frequency_history
		WHERE member_id = $1::uuid
		ORDER BY started_at DESC, created_at DESC`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Frequency{}
	for rows.Next() {
		var f Frequency
		if err := rows.Scan(&f.ID, &f.MemberID, &f.Frequency, &f.StartedAt, &f.EndedAt, &f.Notes); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// SetFrequency registra uma nova frequencia: fecha a vigente (ended_at) e insere
// a nova como atual. Nunca sobrescreve o historico.
func (r *Repo) SetFrequency(ctx context.Context, tx pgx.Tx, memberID, frequency, startedAt, notes, actorID string) (*Frequency, error) {
	if _, err := tx.Exec(ctx, `
		UPDATE member_frequency_history SET ended_at = COALESCE($2::date, current_date)
		WHERE member_id = $1::uuid AND ended_at IS NULL`, memberID, nullStr(startedAt)); err != nil {
		return nil, err
	}
	var newID string
	err := tx.QueryRow(ctx, `
		INSERT INTO member_frequency_history (tenant_id, branch_id, member_id, frequency, started_at, notes, created_by)
		SELECT m.tenant_id, m.branch_id, m.id, $2, COALESCE($3::date, current_date), $4, $5::uuid
		FROM members m WHERE m.id = $1::uuid
		RETURNING id::text`,
		memberID, frequency, nullStr(startedAt), nullStr(notes), nullStr(actorID)).Scan(&newID)
	if err != nil {
		return nil, err
	}
	var f Frequency
	err = tx.QueryRow(ctx, `
		SELECT id::text, member_id::text, frequency, started_at::text, ended_at::text, notes
		FROM member_frequency_history WHERE id = $1::uuid`, newID).
		Scan(&f.ID, &f.MemberID, &f.Frequency, &f.StartedAt, &f.EndedAt, &f.Notes)
	return &f, err
}

// ---- helpers ----

// ErrKindInUse sinaliza que o tipo de evento nao pode ser excluido.
var ErrKindInUse = errKindInUse{}

type errKindInUse struct{}

func (errKindInUse) Error() string {
	return "nao e possivel excluir: ha eventos usando este tipo. Desative-o."
}

func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04", s)
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
